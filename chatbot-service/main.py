"""
=============================================================================
Civic Connect – Chatbot Service / PulseBot  (Python + FastAPI + WebSocket)
=============================================================================
Connects to: Redis (session history), ChromaDB (RAG vector store), Gemini
Port: 8084

PulseBot — a RAG-powered chatbot that retrieves relevant articles, posts,
and complaints from ChromaDB and feeds them as context to Gemini 2.5 Flash.
=============================================================================
"""

import asyncio
import json
import os
import logging
import time
from pathlib import Path
from datetime import datetime, timezone
from contextlib import asynccontextmanager

import pika
import redis as redis_lib
import google.generativeai as genai
import chromadb
import httpx
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi import Response
from fastapi.responses import JSONResponse
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [chatbot-service] %(message)s",
)
log = logging.getLogger(__name__)

# ── Env ──────────────────────────────────────────────────────────────────────

def env(key: str, default: str = "") -> str:
    return os.environ.get(key, default)

PORT = int(env("PORT", "8084"))
REDIS_HOST = env("REDIS_HOST", "localhost")
REDIS_PORT = int(env("REDIS_PORT", "6379"))
REDIS_PASSWORD = env("REDIS_PASSWORD", "redis_secret_2026")
RABBITMQ_USER = env("RABBITMQ_USER", "civic_rabbit")
RABBITMQ_PASS = env("RABBITMQ_PASS", "rabbit_secret_2026")
RABBITMQ_HOST = env("RABBITMQ_HOST", "localhost")
RABBITMQ_URL = env("RABBITMQ_URL", f"amqp://{RABBITMQ_USER}:{RABBITMQ_PASS}@{RABBITMQ_HOST}:5672/")
GEMINI_API_KEY = env("GEMINI_API_KEY", "mock-api-key")
GROQ_API_KEY = env("GROQ_API_KEY", "")
GROQ_MODEL = env("GROQ_MODEL", "llama-3.3-70b-versatile")
CHROMADB_HOST = env("CHROMADB_HOST", "localhost")
CHROMADB_PORT = int(env("CHROMADB_PORT", "8000"))
GEMINI_MODEL = env("GEMINI_MODEL", "gemini-2.5-flash")
GROQ_BASE_URL = env("GROQ_BASE_URL", "https://api.groq.com/openai/v1")
RAG_REFRESH_SECONDS = int(env("RAG_REFRESH_SECONDS", "120"))
ENABLE_PERIODIC_INGEST = env("ENABLE_PERIODIC_INGEST", "true").lower() in ("1", "true", "yes")

# Free-tier model allowlist requested by project.
ALLOWED_GEMINI_MODELS = [
    "gemini-2.0-flash",
    "gemini-2.5-flash",
    "gemini-2.5-flash-lite",
    "gemini-2.5-pro",
]

# Internal service URLs for content ingestion
CONTENT_SERVICE_URL = env("CONTENT_SERVICE_URL", "http://content-service:8082")
COMPLAINT_SERVICE_URL = env("COMPLAINT_SERVICE_URL", "http://complaint-service:8083")
ENABLE_STARTUP_INGEST = env("ENABLE_STARTUP_INGEST", "false").lower() in ("1", "true", "yes")

# ── PulseBot System Prompt ───────────────────────────────────────────────────

PULSEBOT_SYSTEM_PROMPT = """You are PulseBot, the AI assistant for CivicConnect (UrbanPulse) — 
a civic engagement platform connecting citizens with local government.

About the app:
- Citizens can register with Aadhar ID and raise geo-tagged complaints about urban issues
- Complaints support image uploads, upvoting/downvoting, and commenting
- Government officials can track and respond to complaints with action updates
- The social feed ranks posts by location proximity, engagement, and recency
- Citizens can follow government officials and read published articles
- Available as a mobile app (Flutter) and web admin portal

Features you can help with:
- Filing complaints about urban issues (potholes, water, sanitation, etc.)
- Checking complaint status and viewing government responses
- Understanding how the social feed works
- Finding nearby government offices
- Reading government articles and announcements
- Following government officials
- Answering questions about government policies, articles, and community issues

Created by Yogasimman.R — 4th year B.Tech IT students at 
College of Engineering Guindy, Anna University, Chennai.

IMPORTANT INSTRUCTIONS:
- Use the CONTEXT section below to answer questions with real data from the platform.
- If context contains relevant articles, posts, or complaints, reference them specifically.
- Be helpful, concise, and professional.
- If asked about something outside the app's scope, politely redirect.
- When referencing government articles, mention the title and publishing authority.
- Format responses with clear structure using bullet points or numbered lists when helpful.
"""

# ── Connections ──────────────────────────────────────────────────────────────

redis_client: redis_lib.Redis | None = None
rabbitmq_conn: pika.BlockingConnection | None = None
chroma_collection: chromadb.Collection | None = None
gemini_model = None
llm_provider = "fallback"
last_ingest_ts = 0.0
ingest_task: asyncio.Task | None = None
_ingest_lock = asyncio.Lock()

# ── Metrics ──────────────────────────────────────────────────────────────────

chatbot_requests_total = Counter(
    "chatbot_requests_total",
    "Total chatbot /ask requests",
    ["source", "status", "has_government"],
)

chatbot_request_latency_seconds = Histogram(
    "chatbot_request_latency_seconds",
    "Latency for chatbot /ask requests",
    ["source"],
)

chatbot_fallback_total = Counter(
    "chatbot_fallback_total",
    "Total fallback responses by reason",
    ["reason"],
)

chatbot_errors_total = Counter(
    "chatbot_errors_total",
    "Total chatbot endpoint errors",
    ["endpoint"],
)


def connect_redis() -> redis_lib.Redis:
    for i in range(1, 31):
        try:
            client = redis_lib.Redis(
                host=REDIS_HOST,
                port=REDIS_PORT,
                password=REDIS_PASSWORD,
                decode_responses=True,
            )
            client.ping()
            log.info("✅ Redis Connected Successfully")
            return client
        except redis_lib.ConnectionError:
            log.info(f"Waiting for Redis... ({i}/30)")
            time.sleep(2)
    raise RuntimeError("Redis connection failed")


def connect_rabbitmq() -> pika.BlockingConnection:
    params = pika.URLParameters(RABBITMQ_URL)
    for i in range(1, 31):
        try:
            conn = pika.BlockingConnection(params)
            log.info("✅ RabbitMQ Connected Successfully")
            return conn
        except pika.exceptions.AMQPConnectionError:
            log.info(f"Waiting for RabbitMQ... ({i}/30)")
            time.sleep(2)
    raise RuntimeError("RabbitMQ connection failed")


def connect_chromadb() -> chromadb.Collection:
    """Connect to ChromaDB and get/create the civic_docs collection."""
    for i in range(1, 16):
        try:
            client = chromadb.HttpClient(host=CHROMADB_HOST, port=CHROMADB_PORT)
            client.heartbeat()
            collection = client.get_or_create_collection(
                name="civic_docs",
                metadata={"hnsw:space": "cosine"},
            )
            log.info(f"✅ ChromaDB Connected – collection has {collection.count()} docs")
            return collection
        except Exception as e:
            log.info(f"Waiting for ChromaDB... ({i}/15) – {e}")
            time.sleep(3)
    log.warning("⚠️ ChromaDB not available – RAG disabled, using system prompt only")
    return None


def init_gemini():
    """Initialize Gemini model."""
    if GEMINI_API_KEY and GEMINI_API_KEY != "mock-api-key":
        genai.configure(api_key=GEMINI_API_KEY)

        preferred = GEMINI_MODEL if GEMINI_MODEL in ALLOWED_GEMINI_MODELS else "gemini-2.5-flash"
        ordered_models = [preferred] + [m for m in ALLOWED_GEMINI_MODELS if m != preferred]

        for model_name in ordered_models:
            try:
                model = genai.GenerativeModel(model_name)
                log.info(f"✅ Gemini initialized with model: {model_name}")
                return model
            except Exception as e:
                log.warning(f"Gemini model init failed for {model_name}: {e}")

        log.warning("⚠️ No allowed Gemini model could be initialized – using fallback responses")
        return None
    log.warning("⚠️ GEMINI_API_KEY not set – using fallback responses")
    return None


async def generate_groq_response(prompt_text: str) -> str:
    """Call Groq Chat Completions API using the OpenAI-compatible endpoint."""
    payload = {
        "model": GROQ_MODEL,
        "messages": [
            {"role": "system", "content": "You are PulseBot, a civic governance assistant. Be accurate and concise."},
            {"role": "user", "content": prompt_text},
        ],
        "temperature": 0.2,
        "max_tokens": 900,
    }
    headers = {
        "Authorization": f"Bearer {GROQ_API_KEY}",
        "Content-Type": "application/json",
    }

    async with httpx.AsyncClient(timeout=40) as client:
        resp = await client.post(f"{GROQ_BASE_URL}/chat/completions", headers=headers, json=payload)
        resp.raise_for_status()
        data = resp.json()
        choices = data.get("choices", [])
        if not choices:
            raise RuntimeError("Groq returned no choices")
        message = choices[0].get("message", {})
        content = (message.get("content") or "").strip()
        if not content:
            raise RuntimeError("Groq returned empty content")
        return content

# ── RAG Pipeline ─────────────────────────────────────────────────────────────

def extract_query_hints(query: str) -> dict:
    q = (query or "").lower()

    issue_map = {
        "roads": ["road", "roads", "pothole", "street", "waterlogging", "drain"],
        "sanitation": ["waste", "garbage", "trash", "clean", "sanitation", "dump"],
        "water": ["water", "supply", "sewage", "drinking water", "pipeline", "overflow"],
        "public_health": ["mosquito", "dengue", "stagnant", "public health", "hygiene"],
        "mobility": ["traffic", "bus", "transport", "footpath", "crossing", "streetlight"],
    }
    dept_map = {
        "public_works": ["public works", "engineering", "road", "drain"],
        "sanitation": ["sanitation", "solid waste", "waste"],
        "water_board": ["water board", "water supply", "sewerage", "metro water"],
        "public_health": ["public health", "health inspector", "mosquito"],
    }

    issue_hint = ""
    for issue, words in issue_map.items():
        if any(w in q for w in words):
            issue_hint = issue
            break

    dept_hint = ""
    for dept, words in dept_map.items():
        if any(w in q for w in words):
            dept_hint = dept
            break

    gov_terms = []
    for term in [
        "chennai", "chennai corporation", "greater chennai corporation", "tn", "tamil nadu",
        "municipality", "corporation", "government",
    ]:
        if term in q:
            gov_terms.append(term)

    preferred_doc_types = []
    if "complaint" in q or "issue" in q:
        preferred_doc_types.append("complaint")
    if "article" in q or "announcement" in q:
        preferred_doc_types.append("article")
    if "policy" in q and "policy" not in preferred_doc_types:
        preferred_doc_types.append("policy")

    wants_recent = any(w in q for w in ["new", "latest", "recent", "today", "just now"])

    return {
        "issue_hint": issue_hint,
        "dept_hint": dept_hint,
        "gov_terms": gov_terms,
        "preferred_doc_types": preferred_doc_types,
        "wants_recent": wants_recent,
        "query": q,
    }


def parse_iso_ts(value: str) -> float:
    if not value:
        return 0.0
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()
    except Exception:
        return 0.0


def retrieve_context(query: str, government_id: int | None = None, top_k: int = 7) -> str:
    """Retrieve relevant documents from ChromaDB for the given query."""
    if not chroma_collection or chroma_collection.count() == 0:
        return ""

    where_filter = None
    if government_id:
        where_filter = {"government_id": government_id}

    try:
        results = chroma_collection.query(
            query_texts=[query],
            n_results=max(top_k * 3, 12),
            where=where_filter if where_filter else None,
        )
    except Exception:
        # If filter fails (e.g. field doesn't exist), retry without filter
        try:
            results = chroma_collection.query(query_texts=[query], n_results=max(top_k * 3, 12))
        except Exception as e:
            log.error(f"ChromaDB query failed: {e}")
            return ""

    if not results or not results.get("documents") or not results["documents"][0]:
        return ""

    hints = extract_query_hints(query)

    ranked = []
    for idx, (doc, meta) in enumerate(zip(results["documents"][0], results["metadatas"][0])):
        meta = meta or {}
        doc_type = str(meta.get("type", "document")).lower()
        combined = f"{doc}\n{json.dumps(meta)}".lower()
        score = 0

        if hints["preferred_doc_types"] and doc_type in hints["preferred_doc_types"]:
            score += 4
        if hints["issue_hint"] and hints["issue_hint"] in combined:
            score += 3
        if hints["dept_hint"] and hints["dept_hint"] in combined:
            score += 2
        for term in hints["gov_terms"]:
            if term in combined:
                score += 2
        if hints["query"] and any(tok in combined for tok in hints["query"].split()[:5]):
            score += 1
        if hints["wants_recent"]:
            created_at = parse_iso_ts(str(meta.get("created_at", "")))
            if created_at > 0:
                hours_old = max((time.time() - created_at) / 3600.0, 0.0)
                score += max(0.0, 6.0 - min(hours_old / 24.0, 6.0))

        # Preserve retrieval order as tie-breaker
        ranked.append((score, -idx, doc, meta))

    ranked.sort(reverse=True)

    context_parts = []
    for _, _, doc, meta in ranked[:top_k]:
        doc_type = meta.get("type", "document")
        title = meta.get("title", "")
        gov = meta.get("government_name", "")
        dept = meta.get("department", "")
        issue_type = meta.get("issue_type", "")
        header = f"[{doc_type.upper()}] {title}"
        if gov:
            header += f" (from {gov})"
        if dept:
            header += f" [dept: {dept}]"
        if issue_type:
            header += f" [issue: {issue_type}]"
        context_parts.append(f"{header}\n{doc}")

    return "\n\n---\n\n".join(context_parts)


async def generate_response(message: str, history: list, government_id: int | None = None) -> tuple[str, str]:
    """Generate a response using RAG + Gemini.

    Returns: (response_text, response_source)
    """
    await maybe_refresh_rag(message)

    # Retrieve relevant context
    context = retrieve_context(message, government_id)

    if llm_provider == "groq" and GROQ_API_KEY:
        prompt_parts = [PULSEBOT_SYSTEM_PROMPT]
        if context:
            prompt_parts.append(f"\n\nCONTEXT (from CivicConnect database and policy index):\n{context}")

        if history:
            prompt_parts.append("\nRecent conversation:")
            for h in history[-6:]:
                role = "User" if h.get("role") == "user" else "PulseBot"
                prompt_parts.append(f"{role}: {h.get('content', '')}")

        prompt_parts.append(f"\nUser: {message}\nPulseBot:")

        try:
            response_text = await generate_groq_response("\n".join(prompt_parts))
            source = "groq_rag" if context else "groq"
            return response_text, source
        except Exception as e:
            log.error(f"Groq error: {e}")
            fallback_text, fallback_reason = _fallback_response(message)
            chatbot_fallback_total.labels(reason=f"groq_error_{fallback_reason}").inc()
            return fallback_text, "fallback_groq_error"

    if gemini_model:
        # Build prompt with context
        prompt_parts = [PULSEBOT_SYSTEM_PROMPT]
        if context:
            prompt_parts.append(f"\n\nCONTEXT (from CivicConnect database):\n{context}")

        # Add recent history
        if history:
            prompt_parts.append("\nRecent conversation:")
            for h in history[-6:]:
                role = "User" if h.get("role") == "user" else "PulseBot"
                prompt_parts.append(f"{role}: {h.get('content', '')}")

        prompt_parts.append(f"\nUser: {message}\nPulseBot:")

        try:
            response = gemini_model.generate_content("\n".join(prompt_parts))
            source = "gemini_rag" if context else "gemini"
            return response.text.strip(), source
        except Exception as e:
            log.error(f"Gemini error: {e}")
            fallback_text, fallback_reason = _fallback_response(message)
            chatbot_fallback_total.labels(reason=f"gemini_error_{fallback_reason}").inc()
            return fallback_text, "fallback_gemini_error"
    else:
        fallback_text, fallback_reason = _fallback_response(message)
        chatbot_fallback_total.labels(reason=fallback_reason).inc()
        return fallback_text, "fallback_no_gemini"


def ingest_policy_documents() -> int:
    """Load and index Central/TN policy corpus from a local JSON file into ChromaDB."""
    if not chroma_collection:
        return 0

    docs: list[dict] = []
    for policy_file in [
        Path(__file__).with_name("policy_rag_data.json"),
        Path(__file__).with_name("policy_rag_extended.json"),
    ]:
        if not policy_file.exists():
            continue
        try:
            items = json.loads(policy_file.read_text(encoding="utf-8"))
            if isinstance(items, list):
                docs.extend(items)
        except Exception as e:
            log.error(f"Failed to read policy corpus {policy_file.name}: {e}")

    if not docs:
        log.warning("No policy corpus files found for ingestion")
        return 0

    ingested = 0
    for idx, item in enumerate(docs, start=1):
        title = str(item.get("title", "")).strip()
        jurisdiction = str(item.get("jurisdiction", "")).strip().lower() or "india"
        body = str(item.get("content", "")).strip()
        category = str(item.get("category", "policy")).strip().lower()
        if not title or not body:
            continue

        doc_id = item.get("id") or f"policy_{jurisdiction}_{idx}"
        text = f"{title}\n\n{body}"[:3500]
        metadata = {
            "type": "policy",
            "title": title,
            "category": category,
            "government_id": 0,
            "government_name": "Policy Knowledge Base",
            "jurisdiction": jurisdiction,
            "department": str(item.get("department", "urban-governance")).strip().lower(),
            "issue_type": str(item.get("issue_type", category)).strip().lower(),
            "authority_level": str(item.get("authority_level", jurisdiction)).strip().lower(),
        }
        chroma_collection.upsert(ids=[str(doc_id)], documents=[text], metadatas=[metadata])
        ingested += 1

    log.info(f"✅ Ingested {ingested} policy documents into ChromaDB")
    return ingested


def _fallback_response(message: str) -> tuple[str, str]:
    """Keyword-based fallback when Gemini is unavailable.

    Returns: (response_text, fallback_reason)
    """
    msg_lower = message.lower().strip()

    if any(w in msg_lower for w in ["hello", "hi", "hey", "start"]):
        return (
            "Hello! I'm PulseBot, your CivicConnect assistant. "
            "I can help you with filing complaints, checking statuses, "
            "understanding government services, and more. What would you like to do?"
        ), "greeting"
    if any(w in msg_lower for w in ["complaint", "file", "report", "issue"]):
        return (
            "To file a complaint:\n"
            "1. Go to your community page\n"
            "2. Tap 'New Complaint'\n"
            "3. Select a category (pothole, water, sanitation, etc.)\n"
            "4. Add a description and up to 5 photos\n"
            "5. Your GPS location is auto-detected\n\n"
            "Your complaint will be visible to the community and can be upvoted!"
        ), "complaint_help"
    if any(w in msg_lower for w in ["help", "what can", "features"]):
        return (
            "I can help you with:\n"
            "• Filing and tracking complaints\n"
            "• Understanding the social feed\n"
            "• Following government officials\n"
            "• Finding articles and announcements\n"
            "• Learning about government services\n\n"
            "Just ask me anything about CivicConnect!"
        ), "general_help"
    return (
        "I'm PulseBot, your CivicConnect assistant. "
        "I can help with complaints, posts, government services, and more. "
        "Try asking about 'filing a complaint', 'social feed', or 'government officials'. "
        "Type 'help' to see all I can do!"
    ), "default"

# ── Document Ingestion ───────────────────────────────────────────────────────

async def ingest_documents():
    """Fetch articles, posts, and complaints from backend services and ingest into ChromaDB."""
    if not chroma_collection:
        log.warning("ChromaDB not available – skipping ingestion")
        return 0

    global last_ingest_ts
    ingested = 0
    async with httpx.AsyncClient(timeout=30) as client:
        # Ingest articles
        try:
            resp = await client.get(f"{CONTENT_SERVICE_URL}/articles")
            if resp.status_code == 200:
                articles = resp.json() if isinstance(resp.json(), list) else resp.json().get("data", [])
                for art in articles:
                    doc_id = f"article_{art.get('article_id', art.get('id', 0))}"
                    title = art.get("title", "")
                    content = art.get("summary", "") or art.get("content", "")
                    if isinstance(content, dict):
                        blocks = content.get("blocks", [])
                        content = " ".join(b.get("text", "") for b in blocks if isinstance(b, dict))
                    text = f"{title}\n{content}"[:2000]
                    meta = {
                        "type": "article",
                        "title": title,
                        "category": art.get("category", "general"),
                        "government_id": art.get("government_id", 0) or 0,
                        "government_name": art.get("author_gov_name", "") or "",
                        "department": (art.get("author_dept_name", "") or "").lower(),
                        "issue_type": (art.get("category", "general") or "general").lower(),
                        "created_at": str(art.get("created_at", "") or ""),
                    }
                    chroma_collection.upsert(ids=[doc_id], documents=[text], metadatas=[meta])
                    ingested += 1
                log.info(f"Ingested {len(articles)} articles")
        except Exception as e:
            log.error(f"Failed to ingest articles: {e}")

        # Ingest posts
        try:
            resp = await client.get(f"{CONTENT_SERVICE_URL}/posts")
            if resp.status_code == 200:
                posts = resp.json() if isinstance(resp.json(), list) else resp.json().get("data", [])
                for post in posts:
                    doc_id = f"post_{post.get('post_id', post.get('id', 0))}"
                    title = post.get("title", "")
                    body = post.get("content", "")
                    text = f"{title}\n{body}"[:2000]
                    meta = {
                        "type": "post",
                        "title": title,
                        "category": post.get("category", "general"),
                        "government_id": post.get("government_id", 0) or 0,
                        "government_name": post.get("government_name", "") or "",
                        "department": (post.get("department_name", "") or "community").lower(),
                        "issue_type": (post.get("category", "general") or "general").lower(),
                        "created_at": str(post.get("created_at", "") or ""),
                    }
                    chroma_collection.upsert(ids=[doc_id], documents=[text], metadatas=[meta])
                    ingested += 1
                log.info(f"Ingested {len(posts)} posts")
        except Exception as e:
            log.error(f"Failed to ingest posts: {e}")

        # Ingest complaints
        try:
            resp = await client.get(f"{COMPLAINT_SERVICE_URL}/complaints")
            if resp.status_code == 200:
                complaints = resp.json() if isinstance(resp.json(), list) else resp.json().get("data", [])
                for cmp in complaints:
                    doc_id = f"complaint_{cmp.get('id', cmp.get('ID', 0))}"
                    desc = cmp.get("description", "")
                    cat = cmp.get("category", "")
                    status = cmp.get("status", "")
                    text = f"[{cat}] {desc} (Status: {status})"[:2000]
                    meta = {
                        "type": "complaint",
                        "title": f"{cat} complaint",
                        "category": cat,
                        "government_id": cmp.get("government_id", 0) or 0,
                        "government_name": str(cmp.get("government_name", "") or ""),
                        "department": str(cmp.get("department_name", "public_works") or "public_works").lower(),
                        "issue_type": str(cat or "civic_issue").lower(),
                        "created_at": str(cmp.get("created_at", "") or ""),
                    }
                    chroma_collection.upsert(ids=[doc_id], documents=[text], metadatas=[meta])
                    ingested += 1
                log.info(f"Ingested {len(complaints)} complaints")
        except Exception as e:
            log.error(f"Failed to ingest complaints: {e}")

    last_ingest_ts = time.time()
    log.info(f"✅ Total documents ingested: {ingested}")
    return ingested


async def maybe_refresh_rag(message: str = ""):
    """Refresh dynamic RAG sources for community-heavy queries or stale cache."""
    if not chroma_collection:
        return

    now = time.time()
    msg = (message or "").lower()
    force_keywords = ["complaint", "article", "post", "community", "chennai", "corporation", "new"]
    force = any(k in msg for k in force_keywords)
    stale = (now - last_ingest_ts) >= max(RAG_REFRESH_SECONDS, 30)
    if not (force or stale):
        return

    if _ingest_lock.locked():
        return

    async with _ingest_lock:
        await ingest_documents()


async def periodic_ingest_loop():
    while True:
        try:
            await maybe_refresh_rag("community refresh")
        except Exception as e:
            log.error(f"Periodic RAG refresh failed: {e}")
        await asyncio.sleep(max(RAG_REFRESH_SECONDS, 30))


def normalize_article_text(article: dict) -> str:
    title = str(article.get("title", "") or "").strip()
    summary = str(article.get("summary", "") or "").strip()
    content = article.get("content", "")
    if isinstance(content, dict):
        blocks = content.get("blocks", [])
        content = " ".join(b.get("text", "") for b in blocks if isinstance(b, dict))
    body = str(content or "").strip()
    return f"{title}\n{summary}\n{body}"[:3000]


def normalize_post_text(post: dict) -> str:
    title = str(post.get("title", "") or "").strip()
    body = str(post.get("content", "") or "").strip()
    return f"{title}\n{body}"[:2500]


def upsert_event_document(doc_type: str, payload: dict):
    if not chroma_collection:
        return

    if doc_type == "article":
        article_id = payload.get("article_id") or payload.get("id")
        if not article_id:
            return
        doc_id = f"article_{article_id}"
        text = normalize_article_text(payload)
        meta = {
            "type": "article",
            "title": str(payload.get("title", "") or ""),
            "category": str(payload.get("category", "general") or "general"),
            "government_id": payload.get("government_id", 0) or 0,
            "government_name": str(payload.get("author_gov_name", "") or ""),
            "department": str(payload.get("author_dept_name", "") or "").lower(),
            "issue_type": str(payload.get("category", "general") or "general").lower(),
            "created_at": str(payload.get("created_at", "") or ""),
        }
        chroma_collection.upsert(ids=[doc_id], documents=[text], metadatas=[meta])
        return

    if doc_type == "post":
        post_id = payload.get("post_id") or payload.get("id")
        if not post_id:
            return
        doc_id = f"post_{post_id}"
        text = normalize_post_text(payload)
        meta = {
            "type": "post",
            "title": str(payload.get("title", "") or ""),
            "category": str(payload.get("category", "general") or "general"),
            "government_id": payload.get("government_id", 0) or 0,
            "government_name": str(payload.get("government_name", "") or ""),
            "department": str(payload.get("department_name", "") or "community").lower(),
            "issue_type": str(payload.get("category", "general") or "general").lower(),
            "created_at": str(payload.get("created_at", "") or ""),
        }
        chroma_collection.upsert(ids=[doc_id], documents=[text], metadatas=[meta])
        return

    if doc_type == "delete":
        ids = payload.get("ids") or []
        if isinstance(ids, list) and ids:
            try:
                chroma_collection.delete(ids=[str(x) for x in ids])
            except Exception as e:
                log.error(f"Delete ingest event failed: {e}")

# ── WebSocket Manager ────────────────────────────────────────────────────────

class ConnectionManager:
    def __init__(self):
        self.active: dict[str, WebSocket] = {}

    async def connect(self, ws: WebSocket, client_id: str):
        await ws.accept()
        self.active[client_id] = ws
        log.info(f"Client {client_id} connected ({len(self.active)} active)")

    def disconnect(self, client_id: str):
        self.active.pop(client_id, None)
        log.info(f"Client {client_id} disconnected ({len(self.active)} active)")

    async def send(self, client_id: str, message: dict):
        ws = self.active.get(client_id)
        if ws:
            await ws.send_json(message)


manager = ConnectionManager()

# ── FastAPI App ──────────────────────────────────────────────────────────────

@asynccontextmanager
async def lifespan(app: FastAPI):
    global redis_client, rabbitmq_conn, chroma_collection, gemini_model, llm_provider, ingest_task
    log.info("Starting Civic Connect PulseBot Service (RAG)...")
    redis_client = connect_redis()
    rabbitmq_conn = connect_rabbitmq()
    gemini_model = None
    llm_provider = "fallback"

    if GROQ_API_KEY:
        llm_provider = "groq"
        log.info(f"✅ Groq initialized with model: {GROQ_MODEL}")
    else:
        gemini_model = init_gemini()
        if gemini_model:
            llm_provider = "gemini"

    chroma_collection = connect_chromadb()

    if chroma_collection:
        ingest_policy_documents()
        await maybe_refresh_rag("startup")

    # Optional startup ingestion. Disabled by default to avoid heavy cold-start spikes.
    if chroma_collection and ENABLE_STARTUP_INGEST:
        asyncio.create_task(_delayed_ingest())
    if chroma_collection and ENABLE_PERIODIC_INGEST:
        ingest_task = asyncio.create_task(periodic_ingest_loop())

    log.info("✅ All connections established – Connected Successfully")
    yield
    if ingest_task and not ingest_task.done():
        ingest_task.cancel()
    if rabbitmq_conn and not rabbitmq_conn.is_closed:
        rabbitmq_conn.close()
    if redis_client:
        redis_client.close()


async def _delayed_ingest():
    """Wait for other services to be ready, then ingest documents."""
    await asyncio.sleep(15)
    try:
        await ingest_documents()
    except Exception as e:
        log.error(f"Background ingestion failed: {e}")


app = FastAPI(title="Civic Connect PulseBot (RAG)", lifespan=lifespan)


@app.get("/health")
async def health():
    return JSONResponse({
        "status": "healthy",
        "service": "chatbot-service",
        "rag_enabled": chroma_collection is not None,
        "gemini_enabled": gemini_model is not None,
        "llm_provider": llm_provider,
        "groq_enabled": bool(GROQ_API_KEY),
    })


@app.get("/metrics")
async def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)


@app.post("/ask")
async def ask_pulsebot(request: dict):
    """REST endpoint for PulseBot Q&A with RAG."""
    message = request.get("message", "")
    user_id = request.get("user_id", "anonymous")
    government_id = request.get("government_id")

    # Get recent history for context
    history = []
    if redis_client:
        raw = redis_client.lrange(f"pulsebot:{user_id}:history", -10, -1)
        history = [json.loads(item) for item in raw]

    start = time.perf_counter()
    response_source = "unknown"
    try:
        response, response_source = await generate_response(message, history, government_id)
    except Exception as e:
        chatbot_errors_total.labels(endpoint="ask").inc()
        log.error(f"Unexpected /ask error: {e}")
        response, response_source = _fallback_response(message)[0], "fallback_internal_error"
        chatbot_fallback_total.labels(reason="internal_error").inc()

    # Store in Redis session history
    if redis_client:
        redis_client.rpush(f"pulsebot:{user_id}:history", json.dumps({
            "role": "user", "content": message, "timestamp": time.time()
        }))
        redis_client.rpush(f"pulsebot:{user_id}:history", json.dumps({
            "role": "bot", "content": response, "timestamp": time.time()
        }))
        redis_client.ltrim(f"pulsebot:{user_id}:history", -100, -1)

    latency = max(time.perf_counter() - start, 0.0)
    has_government = "true" if government_id is not None else "false"
    chatbot_requests_total.labels(
        source=response_source,
        status="ok",
        has_government=has_government,
    ).inc()
    chatbot_request_latency_seconds.labels(source=response_source).observe(latency)

    return JSONResponse({
        "response": response,
        "response_source": response_source,
        "request_government_id": government_id,
        "latency_ms": int(latency * 1000),
    })


@app.post("/ingest")
async def trigger_ingest():
    """Manually trigger document ingestion into ChromaDB."""
    count = await ingest_documents()
    return JSONResponse({"ingested": count})


@app.post("/ingest/event")
async def ingest_event(request: dict):
    """Lightweight event ingestion endpoint for near real-time RAG updates."""
    doc_type = str(request.get("doc_type", "")).strip().lower()
    payload = request.get("payload") or {}

    if not doc_type or not isinstance(payload, dict):
        return JSONResponse({"error": "doc_type and payload are required"}, status_code=400)

    try:
        async with _ingest_lock:
            upsert_event_document(doc_type, payload)
            return JSONResponse({"status": "ok", "doc_type": doc_type})
    except Exception as e:
        log.error(f"Event ingestion failed: {e}")
        return JSONResponse({"error": "event_ingest_failed"}, status_code=500)


@app.websocket("/ws/{client_id}")
async def websocket_endpoint(ws: WebSocket, client_id: str):
    await manager.connect(ws, client_id)
    if redis_client:
        redis_client.hset(f"pulsebot:{client_id}", "connected_at", str(time.time()))

    try:
        while True:
            data = await ws.receive_text()
            payload = json.loads(data) if data.startswith("{") else {"message": data}
            user_msg = payload.get("message", "")
            government_id = payload.get("government_id")

            # Get recent history
            history = []
            if redis_client:
                redis_client.rpush(f"pulsebot:{client_id}:history", json.dumps({
                    "role": "user", "content": user_msg, "timestamp": time.time(),
                }))
                raw = redis_client.lrange(f"pulsebot:{client_id}:history", -10, -1)
                history = [json.loads(item) for item in raw]

            bot_reply, response_source = await generate_response(user_msg, history, government_id)

            if redis_client:
                redis_client.rpush(f"pulsebot:{client_id}:history", json.dumps({
                    "role": "bot", "content": bot_reply, "timestamp": time.time(),
                }))
                redis_client.ltrim(f"pulsebot:{client_id}:history", -100, -1)

            await manager.send(client_id, {
                "type": "bot_reply",
                "message": bot_reply,
                "response_source": response_source,
            })
    except WebSocketDisconnect:
        manager.disconnect(client_id)
    except Exception as e:
        chatbot_errors_total.labels(endpoint="ws").inc()
        log.error(f"WebSocket error for {client_id}: {e}")
        manager.disconnect(client_id)


@app.get("/history/{client_id}")
async def get_history(client_id: str):
    if not redis_client:
        return JSONResponse({"error": "Redis not available"}, status_code=503)
    raw = redis_client.lrange(f"pulsebot:{client_id}:history", 0, -1)
    history = [json.loads(item) for item in raw]
    return JSONResponse({"client_id": client_id, "history": history})


# ── Entry point ──────────────────────────────────────────────────────────────

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=PORT)
