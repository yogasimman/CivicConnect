-- =============================================================================
-- Civic Connect – PostgreSQL Init Script
-- Creates databases + extensions on startup
-- admin_db is already created by POSTGRES_DB env var, so skip it here
-- =============================================================================

-- Content DB: News feeds, articles, subscriptions
SELECT 'CREATE DATABASE content_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'content_db')\gexec

-- Complaint DB: Complaints, geo-fencing, workflow
SELECT 'CREATE DATABASE complaint_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'complaint_db')\gexec
