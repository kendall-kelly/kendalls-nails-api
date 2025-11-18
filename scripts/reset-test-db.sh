#!/bin/bash

# Reset Test Database Script
# This script drops and recreates the test database for a clean state
# Usage: ./scripts/reset-test-db.sh

set -e

echo "Resetting test database..."

# Check if PostgreSQL container is running
if ! docker ps | grep -q kendalls-nails-postgres; then
    echo "Error: PostgreSQL container is not running"
    echo "Please start it with: docker compose up -d"
    exit 1
fi

# Drop test database if it exists
echo "Dropping kendalls_nails_test database..."
docker exec kendalls-nails-postgres psql -U postgres -c "DROP DATABASE IF EXISTS kendalls_nails_test;" 2>/dev/null

# Create test database
echo "Creating kendalls_nails_test database..."
docker exec kendalls-nails-postgres psql -U postgres -c "CREATE DATABASE kendalls_nails_test;"

echo "✓ Test database has been reset"
echo ""
echo "You can now run tests with a clean database:"
echo "  go test ./..."
