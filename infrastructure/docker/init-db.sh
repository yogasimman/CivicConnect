#!/bin/bash
set -e

# =============================================================================
# Civic Connect – PostgreSQL Init Script
# admin_db is already created by POSTGRES_DB env var
# This script creates content_db and complaint_db + extensions
# =============================================================================

echo "Creating additional databases..."

# Create content_db
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE content_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'content_db')\gexec
EOSQL

# Create complaint_db
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE complaint_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'complaint_db')\gexec
EOSQL

echo "Databases created. Setting up extensions..."

# Extensions on admin_db
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "admin_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOSQL

# Extensions on content_db
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "content_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOSQL

# Extensions on complaint_db
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "complaint_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE EXTENSION IF NOT EXISTS postgis;
    CREATE EXTENSION IF NOT EXISTS postgis_topology;
EOSQL

echo "All databases and extensions initialized successfully."
