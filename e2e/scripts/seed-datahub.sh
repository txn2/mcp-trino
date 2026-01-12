#!/bin/bash
# Seed DataHub with E-commerce Metadata
#
# Usage: ./scripts/seed-datahub.sh
#
# This script ingests the e-commerce database metadata into DataHub
# using the DataHub CLI (datahub). Requires the full environment
# to be running (./scripts/start-full.sh).

set -e

# Change to e2e directory
cd "$(dirname "$0")/.."

DATAHUB_GMS_URL="http://localhost:8081"

echo "Seeding DataHub with E-commerce Metadata..."
echo "============================================"
echo ""

# Check if DataHub is running
if ! curl -s "${DATAHUB_GMS_URL}/health" > /dev/null 2>&1; then
    echo "Error: DataHub GMS is not running."
    echo "Please start the full environment first: ./scripts/start-full.sh"
    exit 1
fi

# Check if datahub CLI is installed
if ! command -v datahub &> /dev/null; then
    echo "Installing DataHub CLI..."
    pip install --quiet 'acryl-datahub[postgres]'
fi

# Create ingestion recipe
cat > /tmp/datahub-recipe.yaml << 'EOF'
source:
  type: postgres
  config:
    host_port: localhost:5432
    database: ecommerce
    username: ecommerce
    password: ecommerce
    schema_pattern:
      allow:
        - "ecommerce"
    include_tables: true
    include_views: true
    profiling:
      enabled: false

transformers:
  - type: simple_add_dataset_ownership
    config:
      owners:
        - urn:li:corpuser:datahub
      ownership_type: DATAOWNER

  - type: pattern_add_dataset_tags
    config:
      tag_pattern:
        rules:
          ".*customers.*": ["urn:li:tag:pii", "urn:li:tag:gdpr-relevant"]
          ".*orders.*": ["urn:li:tag:transactional"]
          ".*products.*": ["urn:li:tag:master-data"]
          ".*revenue.*": ["urn:li:tag:analytics", "urn:li:tag:kpi"]

sink:
  type: datahub-rest
  config:
    server: "http://localhost:8081"
EOF

echo "Running DataHub ingestion..."
datahub ingest -c /tmp/datahub-recipe.yaml

echo ""
echo "Creating glossary terms..."

# Create glossary terms using GraphQL
curl -s -X POST "${DATAHUB_GMS_URL}/api/graphql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation createGlossaryTerm($input: CreateGlossaryEntityInput!) { createGlossaryTerm(input: $input) }",
    "variables": {
      "input": {
        "name": "SKU",
        "description": "Stock Keeping Unit - A unique identifier assigned to each distinct product for inventory management."
      }
    }
  }' > /dev/null

curl -s -X POST "${DATAHUB_GMS_URL}/api/graphql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation createGlossaryTerm($input: CreateGlossaryEntityInput!) { createGlossaryTerm(input: $input) }",
    "variables": {
      "input": {
        "name": "PII",
        "description": "Personally Identifiable Information - Any data that could potentially identify a specific individual."
      }
    }
  }' > /dev/null

curl -s -X POST "${DATAHUB_GMS_URL}/api/graphql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation createGlossaryTerm($input: CreateGlossaryEntityInput!) { createGlossaryTerm(input: $input) }",
    "variables": {
      "input": {
        "name": "Gross Revenue",
        "description": "Total revenue from sales before deducting returns, discounts, or allowances."
      }
    }
  }' > /dev/null

echo "Done!"

# Cleanup
rm -f /tmp/datahub-recipe.yaml

echo ""
echo "============================================"
echo "DataHub seeding complete!"
echo "============================================"
echo ""
echo "View metadata at: http://localhost:9002"
echo ""
echo "Datasets ingested:"
echo "  - postgresql.ecommerce.customers"
echo "  - postgresql.ecommerce.products"
echo "  - postgresql.ecommerce.orders"
echo "  - postgresql.ecommerce.order_items"
echo "  - postgresql.ecommerce.daily_revenue"
echo ""
echo "Test mcp-trino with DataHub provider:"
echo "  TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \\"
echo "  TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \\"
echo "  DATAHUB_ENDPOINT=${DATAHUB_GMS_URL}/api/graphql \\"
echo "  go run ./cmd/mcp-trino"
