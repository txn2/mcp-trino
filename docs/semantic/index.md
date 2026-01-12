# Semantic Layer

AI agents query data more reliably when they understand organizational context: not just table structures, but which datasets are production-ready, what business terms mean, and which columns require careful handling.

mcp-trino's semantic layer integrates with metadata catalogs to surface this context alongside query results.

## How It Works

When you describe a table using `trino_describe_table`, the semantic layer enriches the response with metadata from your configured provider:

```
> Describe the customers table

TABLE: analytics.customers

**Description:** Core customer master data containing all registered users
and their subscription status.

**Owners:** Jane Smith (Data Steward), Platform Team (Technical Owner)
**Domain:** Customer
**Tags:** production, verified

| Column          | Type    | Description                    | Tags          |
|-----------------|---------|--------------------------------|---------------|
| customer_id     | VARCHAR | Unique customer identifier     |               |
| email           | VARCHAR | Customer email address         | **SENSITIVE** |
| subscription    | VARCHAR | Current subscription tier      |               |
| created_at      | DATE    | Account creation timestamp     |               |
```

## Available Metadata

| Type | Description |
|------|-------------|
| **Descriptions** | Business-friendly explanations of tables and columns |
| **Ownership** | Data stewards and technical owners with roles |
| **Tags** | Classification labels attached to assets |
| **Domains** | Business domain assignments (e.g., Customer, Finance) |
| **Glossary Terms** | Links to formal business term definitions |
| **Data Quality** | Freshness scores and quality metrics |
| **Sensitivity** | PII and sensitive data markers at column level |
| **Deprecation** | Warnings with replacement guidance |
| **Lineage** | Upstream and downstream data dependencies |

## Providers

mcp-trino supports multiple metadata sources:

| Provider | Use Case |
|----------|----------|
| [DataHub](datahub.md) | Enterprise metadata catalog with GraphQL API |
| [Static Files](static.md) | YAML or JSON files for simple deployments |
| [Custom](custom.md) | Implement the `Provider` interface for any catalog |

## Zero Overhead When Not Configured

The semantic layer is entirely optional. When no provider is configured:

- No external connections are made
- No additional latency is introduced
- Tool output remains unchanged

This enables gradual adoption. Start with the standalone server, then add semantic context when ready.

## Quick Start

### Option 1: Static File

Create a metadata file:

```yaml
# semantic.yaml
tables:
  - catalog: analytics
    schema: public
    table: customers
    description: Core customer master data
    owners:
      - name: Jane Smith
        type: user
        role: Data Steward
    columns:
      email:
        description: Customer email address
        sensitive: true
```

Configure mcp-trino:

```bash
export SEMANTIC_STATIC_FILE=/path/to/semantic.yaml
```

### Option 2: DataHub

Connect to your DataHub instance:

```bash
export DATAHUB_ENDPOINT=https://datahub.example.com/api/graphql
export DATAHUB_TOKEN=your_token
```

## Caching

Semantic metadata is cached to minimize latency and reduce load on your metadata catalog. See [Caching](caching.md) for configuration options.

## Next Steps

- [DataHub Provider](datahub.md) - Connect to DataHub
- [Static Provider](static.md) - Use YAML/JSON files
- [Custom Providers](custom.md) - Build your own provider
- [Caching](caching.md) - Configure cache behavior
