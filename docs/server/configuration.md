# Configuration

Configure the mcp-trino server using environment variables or configuration files.

## Connection Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `TRINO_HOST` | Trino server hostname | `localhost` |
| `TRINO_PORT` | Server port | `443` (SSL) / `8080` |
| `TRINO_USER` | Authentication username | (required) |
| `TRINO_PASSWORD` | Authentication password | (optional) |
| `TRINO_CATALOG` | Default catalog | `memory` |
| `TRINO_SCHEMA` | Default schema | `default` |
| `TRINO_SSL` | Enable HTTPS | `true` for remote hosts |
| `TRINO_SSL_VERIFY` | Verify SSL certificates | `true` |
| `TRINO_TIMEOUT` | Query timeout (seconds) | `120` |
| `TRINO_SOURCE` | Client identifier | `mcp-trino` |

### Example: Remote Server

```bash
export TRINO_HOST=trino.example.com
export TRINO_USER=analyst
export TRINO_PASSWORD=secret
export TRINO_CATALOG=hive
export TRINO_SCHEMA=default
```

### Example: Local Development

```bash
export TRINO_HOST=localhost
export TRINO_PORT=8080
export TRINO_USER=admin
export TRINO_SSL=false
export TRINO_CATALOG=memory
```

## Extensions

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_TRINO_EXT_READONLY` | Block write operations | `true` |
| `MCP_TRINO_EXT_ERRORS` | Add helpful hints to errors | `true` |
| `MCP_TRINO_EXT_LOGGING` | Enable request logging | `false` |
| `MCP_TRINO_EXT_METRICS` | Enable metrics collection | `false` |
| `MCP_TRINO_EXT_QUERYLOG` | Log all SQL queries | `false` |
| `MCP_TRINO_EXT_METADATA` | Add execution stats to results | `false` |

### Read-Only Mode

Enabled by default. Blocks these statements:

- INSERT, UPDATE, DELETE
- DROP, CREATE, ALTER
- TRUNCATE, MERGE

To allow write operations:

```bash
export MCP_TRINO_EXT_READONLY=false
```

!!! warning
    Only disable read-only mode when necessary and with appropriate Trino permissions.

### Logging

Enable structured JSON logging:

```bash
export MCP_TRINO_EXT_LOGGING=true
```

Output:
```json
{"event":"request_start","tool":"trino_query","request_id":"abc123"}
{"event":"request_success","tool":"trino_query","duration_ms":150}
```

## File-Based Configuration

For production deployments, use a YAML configuration file:

```yaml
# /etc/mcp-trino/config.yaml
trino:
  host: trino.example.com
  port: 443
  user: ${TRINO_USER}           # Environment variable expansion
  password: ${TRINO_PASSWORD}
  catalog: hive
  schema: default
  ssl: true
  ssl_verify: true
  timeout: 120s

toolkit:
  default_limit: 1000
  max_limit: 10000
  default_timeout: 120s
  max_timeout: 300s

extensions:
  readonly: true
  logging: true
  metrics: false
  errors: true
```

Load with:

```bash
export MCP_TRINO_CONFIG=/etc/mcp-trino/config.yaml
export TRINO_USER=service_account
export TRINO_PASSWORD=secret
mcp-trino
```

### Configuration Precedence

1. **Environment variables** (highest priority)
2. **Configuration file**
3. **Defaults** (lowest priority)

This allows base settings in a config file with secrets from environment variables.

## Tool Descriptions

Customize the description that AI agents see for each tool. This is useful for adding domain-specific context (e.g., "Query the retail analytics warehouse") so the agent understands what data is available.

Add a `descriptions` map under `toolkit` in your config file:

```yaml
toolkit:
  default_limit: 1000
  max_limit: 10000
  descriptions:
    trino_query: "Query the retail analytics warehouse. Tables use the hive.sales schema."
    trino_describe_table: "Describe tables in the retail data warehouse."
```

Any tool not listed keeps its default description. The available tool name keys are:

| Key | Tool |
|-----|------|
| `trino_query` | Execute SQL queries |
| `trino_explain` | Get query execution plans |
| `trino_list_catalogs` | List available catalogs |
| `trino_list_schemas` | List schemas in a catalog |
| `trino_list_tables` | List tables in a schema |
| `trino_describe_table` | Describe table columns and types |
| `trino_list_connections` | List configured server connections |

## Tool Annotations

All tools include MCP behavioral annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`) by default. These help AI agents understand tool side effects without executing them.

Schema-browsing tools are marked read-only and idempotent. `trino_query` is marked non-destructive but not read-only (since the SQL content varies). See the [Extensibility guide](../library/extensibility.md#tool-annotations) for the full default table and how to override annotations via the Go API.

## Docker Configuration

### Environment Variables

```bash
docker run --rm -i \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=analyst \
  -e TRINO_PASSWORD=secret \
  -e MCP_TRINO_EXT_LOGGING=true \
  ghcr.io/txn2/mcp-trino:latest
```

### Config File Mount

```bash
docker run --rm -i \
  -v /path/to/config.yaml:/etc/mcp-trino/config.yaml \
  -e MCP_TRINO_CONFIG=/etc/mcp-trino/config.yaml \
  -e TRINO_PASSWORD=secret \
  ghcr.io/txn2/mcp-trino:latest
```

## Next Steps

- [Tools](tools.md) - Learn about available tools
- [Multi-Server](multi-server.md) - Connect multiple Trino clusters
- [Security Reference](../reference/security.md) - Security best practices
