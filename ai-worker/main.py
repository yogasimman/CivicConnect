"""
=============================================================================
Civic Connect – AI Worker  (Python + RabbitMQ Consumer + gRPC Server)
=============================================================================
Connects to: RabbitMQ (consumer), Redis (chat storage), Content Service (gRPC)
Ports: 50052 (gRPC – ChatService + AssistantService)

Domains: AI Chat Threads (conversation management), AI Assistant Q&A,
         Content Summarization (RabbitMQ), Complaint Analysis (RabbitMQ)
=============================================================================
"""

import json
import os
import time
import uuid
import logging
import threading
from concurrent import futures

import pika
import grpc
import redis as redis_lib

# Generated proto stubs (built at Docker image time)
import summary_pb2
import summary_pb2_grpc
import chat_pb2
import chat_pb2_grpc

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [ai-worker] %(message)s",
)
log = logging.getLogger(__name__)

# ── Env ──────────────────────────────────────────────────────────────────────

def env(key: str, default: str = "") -> str:
    return os.environ.get(key, default)

RABBITMQ_USER = env("RABBITMQ_USER", "civic_rabbit")
RABBITMQ_PASS = env("RABBITMQ_PASS", "rabbit_secret_2026")
RABBITMQ_HOST = env("RABBITMQ_HOST", "localhost")
RABBITMQ_URL  = env("RABBITMQ_URL", f"amqp://{RABBITMQ_USER}:{RABBITMQ_PASS}@{RABBITMQ_HOST}:5672/")
CONTENT_GRPC  = env("CONTENT_SERVICE_GRPC", "localhost:50051")
REDIS_HOST    = env("REDIS_HOST", "localhost")
REDIS_PORT    = int(env("REDIS_PORT", "6379"))
REDIS_PASSWORD = env("REDIS_PASSWORD", "redis_secret_2026")
GEMINI_API_KEY = env("GEMINI_API_KEY", "mock-api-key")
GRPC_PORT      = env("AI_GRPC_PORT", "50052")

# ── Redis Connection ─────────────────────────────────────────────────────────

redis_client = None

def connect_redis():
    global redis_client
    for i in range(1, 31):
        try:
            redis_client = redis_lib.Redis(
                host=REDIS_HOST,
                port=REDIS_PORT,
                password=REDIS_PASSWORD,
                decode_responses=True,
            )
            redis_client.ping()
            log.info("✅ Redis Connected Successfully")
            return
        except redis_lib.ConnectionError:
            log.info(f"Waiting for Redis... ({i}/30)")
            time.sleep(2)
    raise RuntimeError("Redis connection failed")

# ── Mock LLM API ─────────────────────────────────────────────────────────────

def mock_llm_chat(messages: list) -> str:
    """Simulates an LLM chat response (replace with Gemini/Mistral in production)."""
    log.info("🤖 Mock LLM – generating chat response...")
    time.sleep(0.3)
    last_msg = messages[-1].get("content", "") if messages else ""
    return (
        f"Thank you for your message. I understand you're asking about: "
        f"'{last_msg[:80]}'. As a civic engagement assistant, I can help you with "
        f"complaints, government services, community updates, and more. "
        f"How can I assist you further?"
    )


def mock_llm_summarize(text: str) -> str:
    """Simulates an LLM summarization call."""
    log.info("🤖 Mock LLM – generating summary...")
    time.sleep(0.3)
    words = text.split()
    if len(words) <= 20:
        return text
    return f"[AI Summary] {' '.join(words[:20])}..."


def mock_llm_analyze_complaint(data: dict) -> str:
    """Simulates complaint analysis."""
    log.info("🤖 Mock LLM – analyzing complaint...")
    time.sleep(0.2)
    category = data.get("category", "unknown")
    return (
        f"[AI Analysis] Category: {category}. "
        f"Priority recommendation: {'high' if category in ('pothole', 'water', 'sewage') else 'medium'}. "
        f"Suggested department: Public Works. "
        f"Estimated resolution time: 48-72 hours."
    )


def mock_llm_assistant(query: str) -> str:
    """Simulates an assistant Q&A response."""
    log.info("🤖 Mock LLM – assistant Q&A...")
    time.sleep(0.3)
    return (
        f"Here's what I found regarding your query: '{query[:60]}'. "
        f"For detailed information, please check the relevant section of the app "
        f"or contact your local government office."
    )

# ── Chat Thread Storage (Redis) ──────────────────────────────────────────────

def get_thread_id(user_id: str) -> int:
    """Allocate a new auto-incrementing thread ID for a user."""
    key = f"user:{user_id}:thread_counter"
    return redis_client.incr(key)


def store_message(user_id: str, thread_id: str, role: str, content: str):
    """Store a message in a chat thread."""
    key = f"chat:{user_id}:{thread_id}"
    msg = json.dumps({
        "role": role,
        "content": content,
        "timestamp": time.time(),
    })
    redis_client.rpush(key, msg)
    # Set thread metadata
    meta_key = f"chat:{user_id}:{thread_id}:meta"
    if not redis_client.exists(meta_key):
        title = content[:30] if content else "New conversation"
        redis_client.hset(meta_key, mapping={
            "title": title,
            "created_at": str(time.time()),
        })
    redis_client.hset(meta_key, "updated_at", str(time.time()))


def get_conversation_history(user_id: str, thread_id: str) -> list:
    """Get all messages in a thread."""
    key = f"chat:{user_id}:{thread_id}"
    raw = redis_client.lrange(key, 0, -1)
    return [json.loads(m) for m in raw]


def list_conversations(user_id: str) -> list:
    """List all conversation threads for a user."""
    pattern = f"chat:{user_id}:*:meta"
    keys = redis_client.keys(pattern)
    conversations = []
    for key in keys:
        parts = key.split(":")
        thread_id = parts[2]
        meta = redis_client.hgetall(key)
        conversations.append({
            "thread_id": thread_id,
            "title": meta.get("title", "Untitled"),
            "created_at": meta.get("created_at"),
            "updated_at": meta.get("updated_at"),
        })
    # Sort by updated_at descending
    conversations.sort(key=lambda x: float(x.get("updated_at", 0)), reverse=True)
    return conversations

# ── gRPC Service Implementation ──────────────────────────────────────────────

class ChatServicer(chat_pb2_grpc.ChatServiceServicer):
    """gRPC ChatService — threaded AI conversations."""

    def GetThreadId(self, request, context):
        user_id = request.user_id
        thread_id = get_thread_id(user_id)
        return chat_pb2.ThreadIdResponse(thread_id=str(thread_id))

    def Generate(self, request, context):
        user_id = request.user_id
        thread_id = request.thread_id
        user_message = request.message

        # Store user message
        store_message(user_id, thread_id, "user", user_message)

        # Get history for context
        history = get_conversation_history(user_id, thread_id)
        messages = [{"role": m["role"], "content": m["content"]} for m in history]

        # Generate AI response
        ai_response = mock_llm_chat(messages)

        # Store AI response
        store_message(user_id, thread_id, "assistant", ai_response)

        return chat_pb2.ChatResponse(response=ai_response)

    def ConversationHistory(self, request, context):
        history = get_conversation_history(request.user_id, request.thread_id)
        messages = []
        for m in history:
            messages.append(chat_pb2.ChatMessage(
                role=m["role"],
                content=m["content"],
                timestamp=str(m.get("timestamp", "")),
            ))
        return chat_pb2.ConversationHistoryResponse(messages=messages)

    def ListConversations(self, request, context):
        convos = list_conversations(request.user_id)
        threads = []
        for c in convos:
            threads.append(chat_pb2.ConversationThread(
                thread_id=c["thread_id"],
                title=c["title"],
                created_at=c.get("created_at", ""),
                updated_at=c.get("updated_at", ""),
            ))
        return chat_pb2.ListConversationsResponse(threads=threads)


class AssistantServicer(chat_pb2_grpc.AssistantServiceServicer):
    """gRPC AssistantService — single-turn Q&A."""

    def AssistantLLM(self, request, context):
        query = request.query
        response = mock_llm_assistant(query)
        return chat_pb2.AssistantResponse(response=response)


def start_grpc_server():
    """Start the gRPC server for Chat and Assistant services."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    chat_pb2_grpc.add_ChatServiceServicer_to_server(ChatServicer(), server)
    chat_pb2_grpc.add_AssistantServiceServicer_to_server(AssistantServicer(), server)
    server.add_insecure_port(f"0.0.0.0:{GRPC_PORT}")
    server.start()
    log.info(f"✅ gRPC server listening on :{GRPC_PORT} (ChatService + AssistantService)")
    return server

# ── gRPC Client (Content Service) ────────────────────────────────────────────

def send_summary_via_grpc(post_id: int, summary_text: str) -> bool:
    """Send the AI-generated summary to content-service via gRPC."""
    try:
        channel = grpc.insecure_channel(CONTENT_GRPC)
        stub = summary_pb2_grpc.ContentUpdaterStub(channel)
        request = summary_pb2.SummaryRequest(
            post_id=post_id,
            summary_text=summary_text,
        )
        response = stub.UpdateSummary(request, timeout=10)
        log.info(f"gRPC response: success={response.success}, message={response.message}")
        return response.success
    except grpc.RpcError as e:
        log.error(f"gRPC call failed: {e}")
        return False

# ── RabbitMQ Callbacks ───────────────────────────────────────────────────────

def on_summarize(ch, method, properties, body):
    """Handle ai_summarize queue messages from content-service."""
    try:
        data = json.loads(body)
        post_id = data["post_id"]
        text = data.get("body", "")
        log.info(f"📝 Summarize request for post {post_id}")

        summary = mock_llm_summarize(text)
        send_summary_via_grpc(post_id, summary)

        ch.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as e:
        log.error(f"Summarize handler error: {e}")
        ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)


def on_complaint_analysis(ch, method, properties, body):
    """Handle complaint_analysis queue messages from complaint-service."""
    try:
        data = json.loads(body)
        complaint_id = data.get("complaint_id")
        log.info(f"🔍 Analysis request for complaint {complaint_id}")

        analysis = mock_llm_analyze_complaint(data)
        log.info(f"Analysis result: {analysis}")

        ch.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as e:
        log.error(f"Analysis handler error: {e}")
        ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

# ── Main ─────────────────────────────────────────────────────────────────────

def connect_rabbitmq() -> pika.BlockingConnection:
    """Connect to RabbitMQ with retry."""
    params = pika.URLParameters(RABBITMQ_URL)
    for i in range(1, 31):
        try:
            conn = pika.BlockingConnection(params)
            log.info("✅ RabbitMQ Connected Successfully")
            return conn
        except pika.exceptions.AMQPConnectionError:
            log.info(f"Waiting for RabbitMQ... ({i}/30)")
            time.sleep(2)
    raise RuntimeError("RabbitMQ connection failed after 30 retries")


def main():
    log.info("Starting Civic Connect AI Worker...")

    connect_redis()
    grpc_server = start_grpc_server()

    connection = connect_rabbitmq()
    channel = connection.channel()

    # Declare queues
    channel.queue_declare(queue="ai_summarize", durable=True)
    channel.queue_declare(queue="complaint_analysis", durable=True)

    channel.basic_qos(prefetch_count=1)
    channel.basic_consume(queue="ai_summarize", on_message_callback=on_summarize)
    channel.basic_consume(queue="complaint_analysis", on_message_callback=on_complaint_analysis)

    log.info("✅ All connections established – Connected Successfully")
    log.info("👂 Listening for messages on [ai_summarize, complaint_analysis]...")
    log.info("🔗 gRPC serving ChatService + AssistantService")

    try:
        channel.start_consuming()
    except KeyboardInterrupt:
        channel.stop_consuming()
    finally:
        connection.close()
        grpc_server.stop(0)


if __name__ == "__main__":
    main()
