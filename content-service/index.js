// =============================================================================
// Civic Connect – Content Service  (Node.js + Express + pg + gRPC)
// =============================================================================
// Connects to: PostgreSQL (content_db), RabbitMQ, Redis, MinIO
// Ports: 8082 (HTTP), 50051 (gRPC – ContentUpdater)
//
// Domains: Social Posts (location-ranked feed), Likes, Bookmarks,
//          Comments, Articles (JSONB + full-text search), Media
// =============================================================================

const express = require("express");
const { Pool } = require("pg");
const amqp = require("amqplib");
const Redis = require("ioredis");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require("path");
const cors = require("cors");

// ── Env helpers ─────────────────────────────────────────────────────────────

const env = (key, fallback) => process.env[key] || fallback;

const PORT      = env("PORT", "8082");
const GRPC_PORT = env("GRPC_PORT", "50051");

// ── PostgreSQL ──────────────────────────────────────────────────────────────

const pool = new Pool({
  host:     env("DB_HOST", "localhost"),
  port:     parseInt(env("DB_PORT", "5432")),
  user:     env("DB_USER", "civic_admin"),
  password: env("DB_PASSWORD", "civic_secret_2026"),
  database: env("DB_NAME", "content_db"),
});

async function connectPostgres() {
  for (let i = 1; i <= 30; i++) {
    try {
      const client = await pool.connect();
      await client.query(`
        -- Social Posts
        CREATE TABLE IF NOT EXISTS posts (
          post_id      SERIAL PRIMARY KEY,
          user_id      INTEGER NOT NULL,
          title        VARCHAR(500) NOT NULL,
          content      TEXT NOT NULL,
          category     VARCHAR(100) DEFAULT 'general',
          post_type    VARCHAR(50) DEFAULT 'text',
          location     VARCHAR(200),
          ai_summary   TEXT,
          created_at   TIMESTAMPTZ DEFAULT NOW(),
          updated_at   TIMESTAMPTZ DEFAULT NOW()
        );

        -- Post Media
        CREATE TABLE IF NOT EXISTS multimedia (
          media_id     SERIAL PRIMARY KEY,
          post_id      INTEGER REFERENCES posts(post_id) ON DELETE CASCADE,
          media_type   VARCHAR(50) DEFAULT 'image',
          media_url    TEXT NOT NULL
        );

        -- Likes (one per user per post)
        CREATE TABLE IF NOT EXISTS likes (
          id           SERIAL PRIMARY KEY,
          user_id      INTEGER NOT NULL,
          post_id      INTEGER REFERENCES posts(post_id) ON DELETE CASCADE,
          created_at   TIMESTAMPTZ DEFAULT NOW(),
          UNIQUE(user_id, post_id)
        );

        -- Bookmarks (one per user per post)
        CREATE TABLE IF NOT EXISTS bookmarks (
          id           SERIAL PRIMARY KEY,
          user_id      INTEGER NOT NULL,
          post_id      INTEGER REFERENCES posts(post_id) ON DELETE CASCADE,
          created_at   TIMESTAMPTZ DEFAULT NOW(),
          UNIQUE(user_id, post_id)
        );

        -- Comments on posts
        CREATE TABLE IF NOT EXISTS comments (
          comment_id   SERIAL PRIMARY KEY,
          user_id      INTEGER NOT NULL,
          post_id      INTEGER REFERENCES posts(post_id) ON DELETE CASCADE,
          content      TEXT NOT NULL,
          created_at   TIMESTAMPTZ DEFAULT NOW()
        );

        -- Government Articles (JSONB content, full-text search)
        CREATE TABLE IF NOT EXISTS articles (
          article_id    SERIAL PRIMARY KEY,
          government_id INTEGER NOT NULL,
          category      INTEGER,
          user_id       INTEGER,
          title         VARCHAR(500) NOT NULL,
          summary       TEXT,
          content       JSONB,
          images        TEXT[],
          search_vector TSVECTOR,
          created_at    TIMESTAMPTZ DEFAULT NOW(),
          updated_at    TIMESTAMPTZ DEFAULT NOW()
        );

        -- Full-text search index on articles
        CREATE INDEX IF NOT EXISTS idx_articles_search ON articles USING GIN(search_vector);

        -- Trigger to auto-update search_vector
        CREATE OR REPLACE FUNCTION articles_search_trigger() RETURNS trigger AS $$
        BEGIN
          NEW.search_vector := to_tsvector('english', COALESCE(NEW.title, '') || ' ' || COALESCE(NEW.summary, ''));
          RETURN NEW;
        END
        $$ LANGUAGE plpgsql;

        DROP TRIGGER IF EXISTS trg_articles_search ON articles;
        CREATE TRIGGER trg_articles_search
          BEFORE INSERT OR UPDATE ON articles
          FOR EACH ROW EXECUTE FUNCTION articles_search_trigger();
      `);
      client.release();
      console.log("[content-service] ✅ PostgreSQL Connected Successfully");
      return;
    } catch (err) {
      console.log(`[content-service] Waiting for PostgreSQL... (${i}/30)`);
      await new Promise((r) => setTimeout(r, 2000));
    }
  }
  console.error("[content-service] PostgreSQL connection failed");
  process.exit(1);
}

// ── RabbitMQ ────────────────────────────────────────────────────────────────

let amqpChannel = null;

async function connectRabbitMQ() {
  const user = env("RABBITMQ_USER", "civic_rabbit");
  const pass = env("RABBITMQ_PASS", "rabbit_secret_2026");
  const host = env("RABBITMQ_HOST", "localhost");
  const url  = env("RABBITMQ_URL", `amqp://${user}:${pass}@${host}:5672/`);

  for (let i = 1; i <= 30; i++) {
    try {
      const conn = await amqp.connect(url);
      amqpChannel = await conn.createChannel();
      await amqpChannel.assertQueue("content_updates", { durable: true });
      await amqpChannel.assertQueue("ai_summarize", { durable: true });
      console.log("[content-service] ✅ RabbitMQ Connected Successfully");
      return;
    } catch {
      console.log(`[content-service] Waiting for RabbitMQ... (${i}/30)`);
      await new Promise((r) => setTimeout(r, 2000));
    }
  }
  console.error("[content-service] RabbitMQ connection failed");
  process.exit(1);
}

// ── Redis ───────────────────────────────────────────────────────────────────

let redisClient = null;

async function connectRedis() {
  redisClient = new Redis({
    host:     env("REDIS_HOST", "localhost"),
    port:     parseInt(env("REDIS_PORT", "6379")),
    password: env("REDIS_PASSWORD", "redis_secret_2026"),
    retryStrategy: (times) => (times > 30 ? null : Math.min(times * 200, 3000)),
  });

  return new Promise((resolve, reject) => {
    redisClient.on("connect", () => {
      console.log("[content-service] ✅ Redis Connected Successfully");
      resolve();
    });
    redisClient.on("error", (err) => {
      console.error("[content-service] Redis error:", err.message);
    });
    setTimeout(() => reject(new Error("Redis timeout")), 60000);
  });
}

// ── Location Scoring Helpers ────────────────────────────────────────────────

// Haversine distance in km
function haversineDistance(lat1, lon1, lat2, lon2) {
  const R = 6371;
  const dLat = ((lat2 - lat1) * Math.PI) / 180;
  const dLon = ((lon2 - lon1) * Math.PI) / 180;
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((lat1 * Math.PI) / 180) *
    Math.cos((lat2 * Math.PI) / 180) *
    Math.sin(dLon / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

// Composite score: 50% distance + 30% engagement + 20% recency
function computeRankScore(post, userLat, userLon) {
  // Distance score (closer = higher)
  const dist = haversineDistance(userLat, userLon, post.lat || 0, post.lon || 0);
  const distScore = Math.max(0, 1 - dist / 500); // 500km max range

  // Engagement score (logarithmic)
  const totalEng = (post.like_count || 0) + (post.comment_count || 0) + (post.bookmark_count || 0);
  const engScore = Math.log1p(totalEng) / 10;

  // Recency score (30-day decay)
  const ageMs = Date.now() - new Date(post.created_at).getTime();
  const ageDays = ageMs / (1000 * 60 * 60 * 24);
  const recencyScore = Math.max(0, 1 - ageDays / 30);

  return 0.5 * distScore + 0.3 * Math.min(engScore, 1) + 0.2 * recencyScore;
}

// ── gRPC Server (ContentUpdater) ────────────────────────────────────────────

function startGrpcServer() {
  const PROTO_PATH = path.join(__dirname, "proto", "summary.proto");
  const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
  });
  const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
  const civicconnect = protoDescriptor.civicconnect;

  const server = new grpc.Server();
  server.addService(civicconnect.ContentUpdater.service, {
    UpdateSummary: async (call, callback) => {
      const { post_id, summary_text } = call.request;
      try {
        await pool.query(
          "UPDATE posts SET ai_summary = $1, updated_at = NOW() WHERE post_id = $2",
          [summary_text, post_id]
        );
        console.log(`[content-service] gRPC: Updated summary for post ${post_id}`);
        callback(null, { success: true, message: "Summary updated" });
      } catch (err) {
        callback(null, { success: false, message: err.message });
      }
    },
  });

  server.bindAsync(
    `0.0.0.0:${GRPC_PORT}`,
    grpc.ServerCredentials.createInsecure(),
    (err) => {
      if (err) {
        console.error("[content-service] gRPC bind failed:", err);
        return;
      }
      console.log(`[content-service] ✅ gRPC server listening on :${GRPC_PORT}`);
    }
  );
}

// ── Express HTTP Server ─────────────────────────────────────────────────────

const app = express();
app.use(cors());
app.use(express.json());

app.get("/health", (_req, res) => {
  res.json({ status: "healthy", service: "content-service" });
});

// ── Posts CRUD ───────────────────────────────────────────────────────────────

// Create post
app.post("/posts", async (req, res) => {
  const { user_id, title, content, category, post_type, location } = req.body;
  try {
    const { rows } = await pool.query(
      `INSERT INTO posts (user_id, title, content, category, post_type, location)
       VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
      [user_id, title, content, category || "general", post_type || "text", location]
    );
    // Publish to RabbitMQ for AI summarization
    if (amqpChannel) {
      amqpChannel.sendToQueue(
        "ai_summarize",
        Buffer.from(JSON.stringify({ post_id: rows[0].post_id, body: content })),
        { persistent: true }
      );
    }
    res.status(201).json(rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Get ranked feed (location-based scoring)
app.get("/posts", async (req, res) => {
  const userLat = parseFloat(req.query.lat) || 0;
  const userLon = parseFloat(req.query.lon) || 0;
  try {
    const { rows } = await pool.query(`
      SELECT p.*,
        COALESCE(l.like_count, 0) AS like_count,
        COALESCE(b.bookmark_count, 0) AS bookmark_count,
        COALESCE(cm.comment_count, 0) AS comment_count
      FROM posts p
      LEFT JOIN (SELECT post_id, COUNT(*) AS like_count FROM likes GROUP BY post_id) l ON l.post_id = p.post_id
      LEFT JOIN (SELECT post_id, COUNT(*) AS bookmark_count FROM bookmarks GROUP BY post_id) b ON b.post_id = p.post_id
      LEFT JOIN (SELECT post_id, COUNT(*) AS comment_count FROM comments GROUP BY post_id) cm ON cm.post_id = p.post_id
      ORDER BY p.created_at DESC
      LIMIT 100
    `);

    // Apply composite ranking if user location provided
    if (userLat !== 0 || userLon !== 0) {
      rows.forEach((post) => {
        post.rank_score = computeRankScore(post, userLat, userLon);
      });
      rows.sort((a, b) => b.rank_score - a.rank_score);
    }

    res.json(rows);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Get single post with details
app.get("/posts/:id", async (req, res) => {
  const userId = req.query.user_id;
  try {
    const { rows } = await pool.query(
      `SELECT p.*,
        COALESCE(l.like_count, 0) AS like_count,
        COALESCE(b.bookmark_count, 0) AS bookmark_count,
        COALESCE(cm.comment_count, 0) AS comment_count
      FROM posts p
      LEFT JOIN (SELECT post_id, COUNT(*) AS like_count FROM likes GROUP BY post_id) l ON l.post_id = p.post_id
      LEFT JOIN (SELECT post_id, COUNT(*) AS bookmark_count FROM bookmarks GROUP BY post_id) b ON b.post_id = p.post_id
      LEFT JOIN (SELECT post_id, COUNT(*) AS comment_count FROM comments GROUP BY post_id) cm ON cm.post_id = p.post_id
      WHERE p.post_id = $1`,
      [req.params.id]
    );
    if (!rows.length) return res.status(404).json({ error: "Post not found" });

    const post = rows[0];

    // Get media
    const media = await pool.query("SELECT * FROM multimedia WHERE post_id = $1", [req.params.id]);
    post.media = media.rows;

    // Get user interaction state
    if (userId) {
      const liked = await pool.query("SELECT 1 FROM likes WHERE user_id = $1 AND post_id = $2", [userId, req.params.id]);
      const bookmarked = await pool.query("SELECT 1 FROM bookmarks WHERE user_id = $1 AND post_id = $2", [userId, req.params.id]);
      post.user_liked = liked.rows.length > 0;
      post.user_bookmarked = bookmarked.rows.length > 0;
    }

    // Get comments
    const comments = await pool.query(
      "SELECT * FROM comments WHERE post_id = $1 ORDER BY created_at DESC",
      [req.params.id]
    );
    post.comments = comments.rows;

    res.json(post);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Delete post (owner only)
app.delete("/posts/:id", async (req, res) => {
  const { user_id } = req.body;
  try {
    const result = await pool.query(
      "DELETE FROM posts WHERE post_id = $1 AND user_id = $2 RETURNING post_id",
      [req.params.id, user_id]
    );
    if (!result.rows.length) return res.status(404).json({ error: "Post not found or not owner" });
    res.json({ message: "deleted" });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// ── Likes ───────────────────────────────────────────────────────────────────

app.post("/likes", async (req, res) => {
  const { user_id, post_id } = req.body;
  try {
    await pool.query(
      `INSERT INTO likes (user_id, post_id) VALUES ($1, $2)
       ON CONFLICT (user_id, post_id) DO NOTHING`,
      [user_id, post_id]
    );
    res.json({ message: "liked" });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.delete("/likes", async (req, res) => {
  const { user_id, post_id } = req.body;
  try {
    await pool.query("DELETE FROM likes WHERE user_id = $1 AND post_id = $2", [user_id, post_id]);
    res.json({ message: "unliked" });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// ── Bookmarks ───────────────────────────────────────────────────────────────

app.post("/bookmarks", async (req, res) => {
  const { user_id, post_id } = req.body;
  try {
    await pool.query(
      `INSERT INTO bookmarks (user_id, post_id) VALUES ($1, $2)
       ON CONFLICT (user_id, post_id) DO NOTHING`,
      [user_id, post_id]
    );
    res.json({ message: "bookmarked" });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.delete("/bookmarks", async (req, res) => {
  const { user_id, post_id } = req.body;
  try {
    await pool.query("DELETE FROM bookmarks WHERE user_id = $1 AND post_id = $2", [user_id, post_id]);
    res.json({ message: "unbookmarked" });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// ── Comments ────────────────────────────────────────────────────────────────

app.post("/comments", async (req, res) => {
  const { user_id, post_id, content } = req.body;
  try {
    const { rows } = await pool.query(
      "INSERT INTO comments (user_id, post_id, content) VALUES ($1, $2, $3) RETURNING *",
      [user_id, post_id, content]
    );
    res.status(201).json(rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.get("/comments/:post_id", async (req, res) => {
  try {
    const { rows } = await pool.query(
      "SELECT * FROM comments WHERE post_id = $1 ORDER BY created_at DESC",
      [req.params.post_id]
    );
    res.json(rows);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// ── Articles ────────────────────────────────────────────────────────────────

app.get("/articles", async (req, res) => {
  const { government_id, search } = req.query;
  try {
    let query = "SELECT * FROM articles";
    const params = [];

    if (government_id && search) {
      query += " WHERE government_id = $1 AND search_vector @@ plainto_tsquery('english', $2)";
      params.push(government_id, search);
    } else if (government_id) {
      query += " WHERE government_id = $1";
      params.push(government_id);
    } else if (search) {
      query += " WHERE search_vector @@ plainto_tsquery('english', $1)";
      params.push(search);
    }

    query += " ORDER BY created_at DESC";
    const { rows } = await pool.query(query, params);
    res.json(rows);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.get("/articles/:id", async (req, res) => {
  try {
    const { rows } = await pool.query("SELECT * FROM articles WHERE article_id = $1", [req.params.id]);
    if (!rows.length) return res.status(404).json({ error: "Article not found" });
    res.json(rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.post("/articles", async (req, res) => {
  const { government_id, category, user_id, title, summary, content, images } = req.body;
  try {
    const { rows } = await pool.query(
      `INSERT INTO articles (government_id, category, user_id, title, summary, content, images)
       VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *`,
      [government_id, category, user_id, title, summary, JSON.stringify(content), images]
    );
    res.status(201).json(rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.put("/articles/:id", async (req, res) => {
  const { title, summary, content, images } = req.body;
  try {
    const { rows } = await pool.query(
      `UPDATE articles SET title = COALESCE($1, title), summary = COALESCE($2, summary),
       content = COALESCE($3, content), images = COALESCE($4, images), updated_at = NOW()
       WHERE article_id = $5 RETURNING *`,
      [title, summary, content ? JSON.stringify(content) : null, images, req.params.id]
    );
    if (!rows.length) return res.status(404).json({ error: "Article not found" });
    res.json(rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.delete("/articles/:id", async (req, res) => {
  try {
    const result = await pool.query("DELETE FROM articles WHERE article_id = $1 RETURNING article_id", [req.params.id]);
    if (!result.rows.length) return res.status(404).json({ error: "Article not found" });
    res.json({ message: "deleted" });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// ── Boot ────────────────────────────────────────────────────────────────────

async function main() {
  console.log("[content-service] Starting Civic Connect Content Service...");

  await connectPostgres();
  await connectRabbitMQ();
  await connectRedis();

  console.log("[content-service] ✅ All connections established – Connected Successfully");

  startGrpcServer();

  app.listen(PORT, () => {
    console.log(`[content-service] HTTP server listening on :${PORT}`);
  });
}

main().catch((err) => {
  console.error("[content-service] Fatal:", err);
  process.exit(1);
});
