#!/bin/bash

# Setup Test Database Script
# This script creates a dedicated test database for running tests
# Usage: ./scripts/setup-test-db.sh

set -e

echo "Setting up test database..."

# Check if PostgreSQL container is running
if ! docker ps | grep -q kendalls-nails-postgres; then
    echo "Error: PostgreSQL container is not running"
    echo "Please start it with: docker compose up -d"
    exit 1
fi

# Create test database if it doesn't exist
echo "Creating kendalls_nails_test database..."
docker exec kendalls-nails-postgres psql -U postgres -c "CREATE DATABASE kendalls_nails_test;" 2>/dev/null || echo "Database already exists"

# Verify test database was created
echo "Verifying test database..."
docker exec kendalls-nails-postgres psql -U postgres -c "\l kendalls_nails_test" | grep kendalls_nails_test > /dev/null

if [ $? -eq 0 ]; then
    echo "✓ Test database 'kendalls_nails_test' is ready"
    echo ""
    echo "Test database URL: postgresql://postgres:postgres@localhost:5432/kendalls_nails_test?sslmode=disable"
    echo ""
    echo "You can now run tests with:"
    echo "  go test ./..."
else
    echo "✗ Failed to create test database"
    exit 1
fi
