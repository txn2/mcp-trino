#!/bin/bash
# Start Tier 2: Full E2E Environment (PostgreSQL + Trino + DataHub)
#
# Usage: ./scripts/start-full.sh
#
# This script starts the full test environment including DataHub.
# DataHub startup may take 2-3 minutes due to Elasticsearch and Kafka.

set -e

# Change to e2e directory
cd "$(dirname "$0")/.."

echo "Starting Tier 2 (Full) E2E Environment..."
echo "================================================"
echo "This includes DataHub and may take 2-3 minutes."
echo ""

# Check if full compose file exists
if [ ! -f "docker-compose.full.yml" ]; then
    echo "Error: docker-compose.full.yml not found."
    echo "Please ensure the full compose file exists."
    exit 1
fi

# Start services
docker compose -f docker-compose.yml -f docker-compose.full.yml up -d

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

echo "Waiting for DataHub GMS to be ready (this may take 2-3 minutes)..."
attempt=0
max_attempts=60
until curl -s http://localhost:8081/health > /dev/null 2>&1; do
    sleep 5
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
        echo ""
        echo "Warning: DataHub is taking longer than expected."
        echo "Check logs with: docker compose logs datahub-gms"
        break
    fi
    printf "."
done
echo " Ready!"

echo ""
echo "================================================"
echo "Full E2E Environment is ready!"
echo "================================================"
echo ""
echo "Services:"
echo "  PostgreSQL:      localhost:5432 (user: ecommerce, password: ecommerce)"
echo "  Trino:           http://localhost:8080"
echo "  DataHub UI:      http://localhost:9002"
echo "  DataHub GraphQL: http://localhost:8081/api/graphql"
echo ""
echo "Test mcp-trino with DataHub provider:"
echo "  TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \\"
echo "  TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \\"
echo "  DATAHUB_ENDPOINT=http://localhost:8081/api/graphql \\"
echo "  go run ./cmd/mcp-trino"
echo ""
echo "To seed DataHub with metadata: ./scripts/seed-datahub.sh"
echo "To stop: ./scripts/stop.sh"
