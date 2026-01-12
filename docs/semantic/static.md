# Static Provider

The static provider loads semantic metadata from YAML or JSON files. It's suitable for:

- Small deployments without a metadata catalog
- Development and testing
- Supplementing other providers with local overrides

## Configuration

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `SEMANTIC_STATIC_FILE` | Yes | - | Path to metadata file |
| `SEMANTIC_STATIC_WATCH_INTERVAL` | No | `0` (disabled) | Hot-reload interval |

### Example

```bash
export SEMANTIC_STATIC_FILE=/etc/mcp-trino/semantic.yaml
export SEMANTIC_STATIC_WATCH_INTERVAL=30s
```

## File Format

### YAML

```yaml
# semantic.yaml
tables:
  - catalog: analytics
    schema: public
    table: customers
    description: Core customer master data containing all registered users
    owners:
      - name: Jane Smith
        type: user
        role: Data Steward
      - name: Platform Team
        type: group
        role: Technical Owner
    tags:
      - name: production
        description: Production-ready dataset
      - name: verified
    domain:
      name: Customer
      description: Customer-related data assets
    glossary_terms:
      - urn:glossary:customer
    columns:
      customer_id:
        description: Unique customer identifier
      email:
        description: Customer email address
        sensitive: true
        sensitivity_level: PII
        tags:
          - pii
      subscription:
        description: Current subscription tier
        glossary_terms:
          - urn:glossary:subscription-tier

  - catalog: analytics
    schema: public
    table: orders_legacy
    description: Legacy orders table
    deprecated: true
    deprecation_note: Migrated to iceberg.sales.orders
    replaced_by: iceberg.sales.orders

glossary:
  - urn: urn:glossary:customer
    name: Customer
    definition: An individual or organization that has registered for our services
  - urn: urn:glossary:subscription-tier
    name: Subscription Tier
    definition: The service level associated with a customer account
    related_terms:
      - urn:glossary:customer
```

### JSON

```json
{
  "tables": [
    {
      "catalog": "analytics",
      "schema": "public",
      "table": "customers",
      "description": "Core customer master data",
      "owners": [
        {"name": "Jane Smith", "type": "user", "role": "Data Steward"}
      ],
      "columns": {
        "email": {
          "description": "Customer email address",
          "sensitive": true
        }
      }
    }
  ],
  "glossary": []
}
```

## Schema Reference

### Table Entry

| Field | Type | Description |
|-------|------|-------------|
| `connection` | string | Trino connection name (optional) |
| `catalog` | string | Catalog name (required) |
| `schema` | string | Schema name (required) |
| `table` | string | Table name (required) |
| `description` | string | Business description |
| `owners` | array | List of owners |
| `tags` | array | List of tags |
| `domain` | object | Domain assignment |
| `glossary_terms` | array | URNs of associated terms |
| `deprecated` | boolean | Deprecation flag |
| `deprecation_note` | string | Deprecation explanation |
| `replaced_by` | string | Replacement table identifier |
| `columns` | object | Column metadata (keyed by name) |
| `custom_properties` | object | Additional key-value metadata |

### Owner Entry

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (optional) |
| `name` | string | Display name (required) |
| `type` | string | `user` or `group` (required) |
| `role` | string | Role description (optional) |

### Tag Entry

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Tag name (required) |
| `description` | string | Tag description (optional) |

### Domain Entry

| Field | Type | Description |
|-------|------|-------------|
| `urn` | string | Domain URN (optional) |
| `name` | string | Domain name (required) |
| `description` | string | Domain description (optional) |

### Column Entry

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Column description |
| `tags` | array | Tag names (strings) |
| `glossary_terms` | array | URNs of associated terms |
| `sensitive` | boolean | Sensitivity flag |
| `sensitivity_level` | string | Classification (e.g., PII, PHI) |

### Glossary Entry

| Field | Type | Description |
|-------|------|-------------|
| `urn` | string | Term URN (required) |
| `name` | string | Term name (required) |
| `definition` | string | Term definition |
| `related_terms` | array | URNs of related terms |

## Hot Reload

Enable automatic reloading when the file changes:

```bash
export SEMANTIC_STATIC_WATCH_INTERVAL=30s
```

The provider checks for file modifications at the specified interval and reloads if the file has changed. This enables updating metadata without restarting mcp-trino.

## Multi-Connection Support

For multi-server deployments, specify the connection name:

```yaml
tables:
  - connection: production
    catalog: hive
    schema: analytics
    table: customers
    description: Production customer data

  - connection: staging
    catalog: hive
    schema: analytics
    table: customers
    description: Staging customer data (test only)
```

## Library Usage

```go
import (
    "github.com/txn2/mcp-trino/pkg/semantic/providers/static"
    "github.com/txn2/mcp-trino/pkg/tools"
)

// Create provider from environment
provider, err := static.New(static.FromEnv())
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Or configure programmatically
provider, err := static.New(static.Config{
    FilePath:      "/etc/mcp-trino/semantic.yaml",
    WatchInterval: 30 * time.Second,
})

// Add to toolkit
toolkit := tools.NewToolkit(trinoClient, cfg,
    tools.WithSemanticProvider(provider),
)
```

## Limitations

The static provider does not support:

- **Lineage**: Returns nil for lineage queries
- **Search**: Basic filtering only (by catalog, schema, tags, owner)

For these features, use the DataHub provider or implement a custom provider.
