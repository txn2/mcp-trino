# E2E Test Environment

This directory contains Docker Compose configurations for testing mcp-trino's semantic provider functionality with real infrastructure.

## Quick Start

### Tier 1: Lightweight (Recommended for Development)

Fast startup (~30 seconds) with PostgreSQL and Trino:

```bash
cd e2e
./scripts/start.sh
```

Test mcp-trino:

```bash
TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \
TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \
go run ./cmd/mcp-trino
```

### Tier 2: Full Stack (DataHub Integration)

Complete environment including DataHub for metadata catalog testing:

```bash
cd e2e
./scripts/start-full.sh

# Seed DataHub with metadata (after services are ready)
./scripts/seed-datahub.sh
```

Test mcp-trino with DataHub provider:

```bash
TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \
TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \
DATAHUB_ENDPOINT=http://localhost:8081/api/graphql \
go run ./cmd/mcp-trino
```

## Services

| Service | Tier 1 | Tier 2 | Port | Description |
|---------|--------|--------|------|-------------|
| PostgreSQL | Yes | Yes | 5432 | E-commerce sample database |
| Trino | Yes | Yes | 8080 | Query engine |
| DataHub UI | No | Yes | 9002 | Metadata catalog UI |
| DataHub GMS | No | Yes | 8081 | GraphQL API |

## Sample Data

The e-commerce database includes:

### Tables

- **customers** - Customer master data (PII, GDPR-relevant)
- **products** - Product catalog with pricing
- **orders** - Customer orders
- **order_items** - Order line items
- **daily_revenue** - Analytics view

### Seed Data

- 5 customers
- 10 products across Electronics and Apparel categories
- 10 orders with various statuses
- Sample line items connecting orders to products

### Semantic Metadata

The static provider configuration (`config/semantic/ecommerce.yaml`) includes:

- Table and column descriptions
- Data ownership information
- Tags (pii, gdpr-relevant, financial, etc.)
- Data quality scores
- Glossary terms (SKU, PII, Gross Revenue, etc.)
- Sensitivity classifications

## Directory Structure

```
e2e/
├── docker-compose.yml           # Tier 1: PostgreSQL + Trino
├── docker-compose.full.yml      # Tier 2: + DataHub
├── init/
│   └── postgres/
│       └── 01-ecommerce.sql     # Database schema + seed data
├── config/
│   ├── trino/
│   │   └── catalog/
│   │       └── postgresql.properties
│   └── semantic/
│       └── ecommerce.yaml       # Static semantic metadata
├── scripts/
│   ├── start.sh                 # Start Tier 1
│   ├── start-full.sh            # Start Tier 2
│   ├── stop.sh                  # Stop all services
│   └── seed-datahub.sh          # Ingest metadata into DataHub
└── README.md                    # This file
```

## Usage Examples

### Query via Trino CLI

```bash
# List schemas
docker compose exec trino trino --execute 'SHOW SCHEMAS FROM postgresql'

# List tables
docker compose exec trino trino --execute 'SHOW TABLES FROM postgresql.ecommerce'

# Query customers
docker compose exec trino trino --execute 'SELECT * FROM postgresql.ecommerce.customers LIMIT 5'

# Query daily revenue
docker compose exec trino trino --execute 'SELECT * FROM postgresql.ecommerce.daily_revenue'
```

### Test Semantic Provider

Using the static provider (file-based):

```bash
TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \
TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \
SEMANTIC_FILE=./e2e/config/semantic/ecommerce.yaml \
go run ./cmd/mcp-trino
```

Using the DataHub provider (requires Tier 2):

```bash
TRINO_HOST=localhost TRINO_PORT=8080 TRINO_USER=admin \
TRINO_CATALOG=postgresql TRINO_SCHEMA=ecommerce \
DATAHUB_ENDPOINT=http://localhost:8081/api/graphql \
go run ./cmd/mcp-trino
```

## Stopping the Environment

```bash
# Stop containers (preserves data)
./scripts/stop.sh

# Stop and remove all data
./scripts/stop.sh --clean
```

## Resource Requirements

### Tier 1 (Lightweight)
- 2GB RAM
- 2 CPU cores
- ~30 second startup

### Tier 2 (Full)
- 8GB+ RAM recommended
- 4+ CPU cores
- 2-3 minute startup (due to Elasticsearch/Kafka)

## Troubleshooting

### Services won't start

Check Docker resources:
```bash
docker system info | grep -E "Memory|CPUs"
```

### DataHub takes too long

DataHub requires Elasticsearch and Kafka to be ready. Check logs:
```bash
docker compose logs datahub-gms
docker compose logs elasticsearch
```

### Trino can't connect to PostgreSQL

Ensure PostgreSQL is healthy:
```bash
docker compose exec postgres pg_isready -U ecommerce
```

### Port conflicts

If ports are in use, modify the port mappings in `docker-compose.yml`:
```yaml
ports:
  - "15432:5432"  # Change PostgreSQL port
  - "18080:8080"  # Change Trino port
```
