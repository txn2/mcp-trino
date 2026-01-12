#!/bin/bash
# Start Tier 1: Lightweight E2E Environment (PostgreSQL + Trino)
#
# Usage: ./scripts/start.sh
#
# This script starts the lightweight test environment and waits for
# all services to be healthy before returning.

set -e

# Change to e2e directory
cd "$(dirname "$0")/.."

echo "Starting Tier 1 (Lightweight) E2E Environment..."
echo "================================================"
echo ""

# Start services
docker compose up -d

echo ""
echo "Waiting for PostgreSQL to be ready..."
until docker compose exec -T postgres pg_isready -U ecommerce > /dev/null 2>&1; do
    sleep 1
    printf "."
done
echo " Ready!"

echo "Waiting for Trino to be ready..."
until docker compose exec -T trino trino --execute "SELECT 1" > /dev/null 2>&1; do
    sleep 2
    printf "."
done
echo " Ready!"

echo ""
echo "================================================"
echo "E2E Environment is ready!"
echo "================================================"
echo ""
echo "Services:"
echo "  PostgreSQL: localhost:5432 (user: ecommerce, password: ecommerce)"
echo "  Trino:      http://localhost:8080"
echo ""
echo "Test mcp-trino with:"
echo "  TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \\"
echo "  TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \\"
echo "  go run ./cmd/mcp-trino"
echo ""
echo "Or with static semantic provider:"
echo "  TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \\"
echo "  TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \\"
echo "  SEMANTIC_FILE=./e2e/config/semantic/ecommerce.yaml \\"
echo "  go run ./cmd/mcp-trino"
echo ""
echo "Sample queries via Trino CLI:"
echo "  docker compose exec trino trino --execute 'SHOW SCHEMAS FROM postgresql'"
echo "  docker compose exec trino trino --execute 'SELECT * FROM postgresql.ecommerce.customers LIMIT 5'"
echo ""
echo "To stop: ./scripts/stop.sh"
