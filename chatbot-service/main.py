"""
=============================================================================
Civic Connect – Chatbot Service / PulseBot  (Python + FastAPI + WebSocket)
=============================================================================
Connects to: Redis (chat history), RabbitMQ
Port: 8084

PulseBot — an AI-powered chatbot with knowledge about the CivicConnect/
UrbanPulse app. Uses Gemini 2.5 Flash (mocked for now) with a system
prompt containing app context, features, and creator info.
=============================================================================
"""

import asyncio
import json
import os
import logging
import time
from contextlib import asynccontextmanager

import pika
import redis as redis_lib
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse

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

# ── PulseBot System Prompt ───────────────────────────────────────────────────

PULSEBOT_SYSTEM_PROMPT = """You are PulseBot, the AI assistant for CivicConnect (UrbanPulse) — 
a civic engagement platform connecting citizens with local government.

About the app:
- Citizens can register with Aadhar ID and raise geo-tagged complaints about urban issues
- Complaints support image uploads, upvoting/downvoting, and commenting
- Government officials can track and respond to complaints with action updates
- The social feed ranks posts by location proximity, engagement, and recency
- Citizens can follow government officials and read published articles
- Available as a mobile app (React Native/Flutter) and web admin portal

Features you can help with:
- Filing complaints about urban issues (potholes, water, sanitation, etc.)
- Checking complaint status and viewing government responses
- Understanding how the social feed works
- Finding nearby government offices
- Reading government articles and announcements
- Following government officials

Created by Manish.M and Yogasimman.R — 4th year B.Tech IT students at 
College of Engineering Guindy, Anna University, Chennai.

Always be helpful, concise, and professional. If asked about something 
outside the app's scope, politely redirect to relevant app features."""

# ── Connections ──────────────────────────────────────────────────────────────

redis_client: redis_lib.Redis | None = None
rabbitmq_conn: pika.BlockingConnection | None = None


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

# ── PulseBot Logic ───────────────────────────────────────────────────────────

def pulsebot_response(message: str, history: list) -> str:
    """Generate a PulseBot response (mock Gemini — replace with real API).
    
    In production, this would call Gemini 2.5 Flash with:
    - PULSEBOT_SYSTEM_PROMPT as system context
    - Recent chat history for continuity
    - User's message
    """
    msg_lower = message.lower().strip()

    # Knowledge-based responses matching the product features
    if any(w in msg_lower for w in ["hello", "hi", "hey", "start"]):
        return (
            "Hello! I'm PulseBot, your CivicConnect assistant. "
            "I can help you with filing complaints, checking statuses, "
            "understanding government services, and more. What would you like to do?"
        )
    
    if any(w in msg_lower for w in ["complaint", "file", "report", "issue"]):
        return (
            "To file a complaint:\n"
            "1. Go to your community page\n"
            "2. Tap 'New Complaint'\n"
            "3. Select a category (pothole, water, sanitation, etc.)\n"
            "4. Add a description and up to 5 photos\n"
            "5. Your GPS location is auto-detected, or add manually\n\n"
            "Your complaint will be visible to the community and can be upvoted!"
        )
    
    if any(w in msg_lower for w in ["status", "track", "progress"]):
        return (
            "To check your complaint status:\n"
            "• Go to your community's complaint section\n"
            "• Find your complaint — it shows: Pending, In Progress, or Resolved\n"
            "• Government officials add 'Actions Taken' with completion percentage\n"
            "• Complaints auto-resolve when completion reaches 100%"
        )
    
    if any(w in msg_lower for w in ["feed", "post", "social"]):
        return (
            "The social feed ranks posts using a smart algorithm:\n"
            "• 50% — Geographic proximity (closer to you = higher rank)\n"
            "• 30% — Engagement (likes, comments, bookmarks)\n"
            "• 20% — Recency (newer posts score higher)\n\n"
            "You can like, bookmark, and comment on any post!"
        )
    
    if any(w in msg_lower for w in ["follow", "official", "government"]):
        return (
            "You can follow government officials to stay updated:\n"
            "• Browse the Officials directory\n"
            "• Tap Follow to get their posts in your feed\n"
            "• View their department and position info\n"
            "• See articles published by your local government"
        )
    
    if any(w in msg_lower for w in ["article", "news", "announcement"]):
        return (
            "Government articles are published by local authorities:\n"
            "• Browse articles by your community\n"
            "• Articles support rich content with images\n"
            "• Use search to find specific topics\n"
            "• Each article shows the publishing department"
        )
    
    if any(w in msg_lower for w in ["who made", "creator", "about", "developer", "built"]):
        return (
            "CivicConnect (UrbanPulse) was created by:\n"
            "• Manish.M and Yogasimman.R\n"
            "• 4th year B.Tech IT students\n"
            "• College of Engineering Guindy, Anna University, Chennai\n\n"
            "Built with love for better civic engagement!"
        )
    
    if any(w in msg_lower for w in ["help", "what can", "features"]):
        return (
            "I can help you with:\n"
            "• 📋 Filing and tracking complaints\n"
            "• 📰 Understanding the social feed\n"
            "• 👥 Following government officials\n"
            "• 📄 Finding articles and announcements\n"
            "• 🏛️ Learning about government services\n"
            "• ℹ️ App features and navigation\n\n"
            "Just ask me anything about CivicConnect!"
        )
    
    return (
        "I'm PulseBot, your CivicConnect assistant. "
        "I can help with complaints, posts, government services, and more. "
        "Try asking about 'filing a complaint', 'social feed', or 'government officials'. "
        "Type 'help' to see all I can do!"
    )

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
    global redis_client, rabbitmq_conn
    log.info("Starting Civic Connect PulseBot Service...")
    redis_client = connect_redis()
    rabbitmq_conn = connect_rabbitmq()
    log.info("✅ All connections established – Connected Successfully")
    yield
    if rabbitmq_conn and not rabbitmq_conn.is_closed:
        rabbitmq_conn.close()
    if redis_client:
        redis_client.close()


app = FastAPI(title="Civic Connect PulseBot", lifespan=lifespan)


@app.get("/health")
async def health():
    return JSONResponse({"status": "healthy", "service": "chatbot-service"})


@app.post("/ask")
async def ask_pulsebot(request: dict):
    """REST endpoint for PulseBot Q&A (alternative to WebSocket)."""
    message = request.get("message", "")
    user_id = request.get("user_id", "anonymous")

    # Get recent history for context
    history = []
    if redis_client:
        raw = redis_client.lrange(f"pulsebot:{user_id}:history", -10, -1)
        history = [json.loads(item) for item in raw]

    response = pulsebot_response(message, history)

    # Store in history
    if redis_client:
        redis_client.rpush(f"pulsebot:{user_id}:history", json.dumps({
            "role": "user", "content": message, "timestamp": time.time()
        }))
        redis_client.rpush(f"pulsebot:{user_id}:history", json.dumps({
            "role": "bot", "content": response, "timestamp": time.time()
        }))
        # Keep last 100 messages
        redis_client.ltrim(f"pulsebot:{user_id}:history", -100, -1)

    return JSONResponse({"response": response})


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

            # Get recent history
            history = []
            if redis_client:
                redis_client.rpush(f"pulsebot:{client_id}:history", json.dumps({
                    "role": "user", "content": user_msg, "timestamp": time.time(),
                }))
                raw = redis_client.lrange(f"pulsebot:{client_id}:history", -10, -1)
                history = [json.loads(item) for item in raw]

            bot_reply = pulsebot_response(user_msg, history)

            if redis_client:
                redis_client.rpush(f"pulsebot:{client_id}:history", json.dumps({
                    "role": "bot", "content": bot_reply, "timestamp": time.time(),
                }))
                redis_client.ltrim(f"pulsebot:{client_id}:history", -100, -1)

            await manager.send(client_id, {
                "type": "bot_reply",
                "message": bot_reply,
            })
    except WebSocketDisconnect:
        manager.disconnect(client_id)
    except Exception as e:
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
