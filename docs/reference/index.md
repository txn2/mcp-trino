# Reference

Complete technical reference for mcp-trino.

## Quick Links

| Section | Description |
|---------|-------------|
| [Tools API](tools-api.md) | All tool parameters, responses, and error codes |
| [Configuration](configuration.md) | Environment variables and YAML schema |
| [Security](security.md) | Limits, authentication, and verification |

## Tools

| Tool | Description |
|------|-------------|
| `trino_query` | Execute SQL queries with limits and timeouts |
| `trino_explain` | Get query execution plans |
| `trino_list_catalogs` | List available data catalogs |
| `trino_list_schemas` | List schemas in a catalog |
| `trino_list_tables` | List tables with optional pattern filter |
| `trino_describe_table` | Get column definitions and sample data |
| `trino_list_connections` | List configured server connections |

## Environment Variables

### Connection

| Variable | Default | Description |
|----------|---------|-------------|
| `TRINO_HOST` | `localhost` | Server hostname |
| `TRINO_PORT` | `443`/`8080` | Server port |
| `TRINO_USER` | (required) | Username |
| `TRINO_PASSWORD` | | Password |
| `TRINO_CATALOG` | `memory` | Default catalog |
| `TRINO_SCHEMA` | `default` | Default schema |
| `TRINO_SSL` | `true` | Enable HTTPS |

### Extensions

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_TRINO_EXT_READONLY` | `true` | Block write operations |
| `MCP_TRINO_EXT_LOGGING` | `false` | Enable request logging |
| `MCP_TRINO_EXT_ERRORS` | `true` | Add helpful error hints |

## Limits

| Limit | Default | Maximum |
|-------|---------|---------|
| Row limit | 1000 | 10000 |
| Query timeout | 120s | 300s |

## Blocked Operations (Read-Only Mode)

- INSERT
- UPDATE
- DELETE
- DROP
- CREATE
- ALTER
- TRUNCATE
- MERGE

## Release Verification

All releases include:

- **Checksums** - SHA256 verification
- **SLSA Provenance** - Level 3 build attestation
- **Cosign Signatures** - Keyless verification

```bash
# Verify with Cosign
cosign verify-blob \
  --bundle mcp-trino_*.tar.gz.sigstore.json \
  mcp-trino_*.tar.gz
```
