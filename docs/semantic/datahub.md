# DataHub Provider

The DataHub provider connects mcp-trino to [DataHub](https://datahubproject.io/), an open-source metadata platform. It retrieves table descriptions, ownership, tags, glossary terms, domains, and column-level metadata via DataHub's GraphQL API.

## Configuration

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `DATAHUB_ENDPOINT` | Yes | - | DataHub GraphQL endpoint URL |
| `DATAHUB_TOKEN` | Yes | - | Authentication token |
| `DATAHUB_PLATFORM` | No | `trino` | Data platform name in URNs |
| `DATAHUB_ENVIRONMENT` | No | `PROD` | DataHub environment |
| `DATAHUB_TIMEOUT` | No | `30s` | Request timeout |

### Example

```bash
export DATAHUB_ENDPOINT=https://datahub.example.com/api/graphql
export DATAHUB_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
export DATAHUB_PLATFORM=trino
export DATAHUB_ENVIRONMENT=PROD
```

## URN Mapping

DataHub identifies datasets using URNs (Uniform Resource Names). The provider maps Trino table identifiers to DataHub URNs:

```
Trino: catalog.schema.table
DataHub: urn:li:dataset:(urn:li:dataPlatform:{platform},{catalog}.{schema}.{table},{environment})
```

For example, with default settings:

| Trino Table | DataHub URN |
|-------------|-------------|
| `hive.analytics.customers` | `urn:li:dataset:(urn:li:dataPlatform:trino,hive.analytics.customers,PROD)` |
| `iceberg.sales.orders` | `urn:li:dataset:(urn:li:dataPlatform:trino,iceberg.sales.orders,PROD)` |

### Custom Platform Names

If your DataHub instance uses a different platform name for Trino datasets:

```bash
export DATAHUB_PLATFORM=trino-prod
```

### Multi-Environment

For staging or development environments:

```bash
export DATAHUB_ENVIRONMENT=DEV
```

## Supported Metadata

The provider retrieves the following metadata from DataHub:

### Table-Level

| Metadata | DataHub Source |
|----------|----------------|
| Description | Dataset properties |
| Ownership | Ownership aspect with roles |
| Tags | Global tags with descriptions |
| Glossary Terms | Glossary term associations |
| Domain | Domain assignment |
| Deprecation | Deprecation aspect |

### Column-Level

| Metadata | DataHub Source |
|----------|----------------|
| Description | Schema field description |
| Tags | Field-level tags |
| Glossary Terms | Field-level term associations |
| Sensitivity | Tags containing "pii" or "sensitive" |

## Sensitivity Detection

The provider automatically marks columns as sensitive when they have tags containing:

- `pii`
- `sensitive`

For example, a column with tag `pii-email` or `sensitive-data` will be marked as sensitive in the output.

## Library Usage

```go
import (
    "github.com/txn2/mcp-trino/pkg/semantic/providers/datahub"
    "github.com/txn2/mcp-trino/pkg/tools"
)

// Create provider from environment
provider, err := datahub.New(datahub.FromEnv())
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Or configure programmatically
provider, err := datahub.New(datahub.Config{
    Endpoint:    "https://datahub.example.com/api/graphql",
    Token:       "your-token",
    Platform:    "trino",
    Environment: "PROD",
    Timeout:     30 * time.Second,
})

// Add to toolkit
toolkit := tools.NewToolkit(trinoClient, cfg,
    tools.WithSemanticProvider(provider),
    tools.WithSemanticCache(semantic.DefaultCacheConfig()),
)
```

## Error Handling

The provider follows these conventions:

- Returns `nil` (not an error) when metadata is not found
- Returns errors only for connection or authentication failures
- Logs warnings for malformed URNs or unexpected responses

This ensures that missing metadata doesn't break tool execution.

## Troubleshooting

### Authentication Errors

```
datahub: token is required
```

Ensure `DATAHUB_TOKEN` is set with a valid access token from DataHub.

### Connection Timeouts

```
context deadline exceeded
```

Increase the timeout or check network connectivity:

```bash
export DATAHUB_TIMEOUT=60s
```

### Dataset Not Found

If metadata isn't appearing, verify the URN mapping:

1. Check the platform name matches DataHub's configuration
2. Verify the environment (PROD, DEV, etc.)
3. Confirm the dataset exists in DataHub with the expected URN

Use DataHub's search or browse interface to find the exact URN for a dataset.
