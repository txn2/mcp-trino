#!/bin/bash
# Stop E2E Environment
#
# Usage: ./scripts/stop.sh [--clean]
#
# Options:
#   --clean    Also remove volumes (database data)

set -e

# Change to e2e directory
cd "$(dirname "$0")/.."

echo "Stopping E2E Environment..."

if [ "$1" == "--clean" ]; then
    echo "Removing containers and volumes..."
    docker compose -f docker-compose.yml -f docker-compose.full.yml down -v 2>/dev/null || docker compose down -v
else
    echo "Stopping containers (data preserved in volumes)..."
    docker compose -f docker-compose.yml -f docker-compose.full.yml down 2>/dev/null || docker compose down
fi

echo ""
echo "E2E Environment stopped."
echo ""
echo "To restart: ./scripts/start.sh"
echo "To clean data: ./scripts/stop.sh --clean"
