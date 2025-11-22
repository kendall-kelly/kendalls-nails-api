#!/bin/bash
set -e

# Create test database if it doesn't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE kendalls_nails_test'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kendalls_nails_test')\gexec
EOSQL

echo "Test database 'kendalls_nails_test' is ready"
