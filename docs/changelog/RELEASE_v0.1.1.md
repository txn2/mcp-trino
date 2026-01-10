# v0.1.1 - Initial Release

The first public release of mcp-trino, a Model Context Protocol server for Trino data warehouses.

## Features

### MCP Tools
- **trino_query** - Execute SQL queries with configurable limits, timeouts, and output formats (JSON, CSV, Markdown)
- **trino_explain** - Analyze query execution plans (logical, distributed, IO, validate)
- **trino_list_catalogs** - Discover available data catalogs
- **trino_list_schemas** - Browse schemas within a catalog
- **trino_list_tables** - List tables with optional pattern filtering
- **trino_describe_table** - Inspect table columns and sample data
- **trino_list_connections** - View configured server connections

### Multi-Server Support
- Connect to multiple Trino servers from a single installation
- Configure additional servers via JSON in `TRINO_ADDITIONAL_SERVERS`
- Optional `connection` parameter on all query tools
- Credential inheritance from primary server

### Extensibility
- **Composable library** - Import `pkg/tools` and `pkg/client` into custom MCP servers
- **Middleware system** - Add authentication, logging, tenant isolation
- **Query interceptors** - Transform SQL before execution
- **Result transformers** - Modify query results

### Built-in Extensions
- **Read-only mode** (enabled by default) - Blocks INSERT, UPDATE, DELETE
- **Error help** - Adds helpful hints to Trino error messages
- **Query logging** - Audit all SQL queries
- **Metrics collection** - Track query performance
- **Metadata footer** - Add execution stats to results

### Security
- Configurable query limits (default: 1000 rows, max: 10000)
- Query timeouts (default: 120s)
- SSL/TLS with certificate verification
- SLSA Level 3 provenance on all release artifacts
- Cosign signatures for verification

## Installation

### Homebrew (macOS)
```bash
brew install txn2/tap/mcp-trino
```

### Claude Desktop
Download the `.mcpb` bundle for your platform from the [releases page](https://github.com/txn2/mcp-trino/releases/tag/v0.1.1):
- macOS Apple Silicon: `mcp-trino_0.1.1_darwin_arm64.mcpb`
- macOS Intel: `mcp-trino_0.1.1_darwin_amd64.mcpb`
- Windows: `mcp-trino_0.1.1_windows_amd64.mcpb`

### Claude Code CLI
```bash
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  -- mcp-trino
```

### Docker
```bash
docker pull ghcr.io/txn2/mcp-trino:v0.1.1
```

### Go Install
```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@v0.1.1
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `TRINO_HOST` | Server hostname | `localhost` |
| `TRINO_PORT` | Server port | `443` (SSL) / `8080` |
| `TRINO_USER` | Username | (required) |
| `TRINO_PASSWORD` | Password | (optional) |
| `TRINO_CATALOG` | Default catalog | `memory` |
| `TRINO_SCHEMA` | Default schema | `default` |
| `TRINO_SSL` | Enable HTTPS | `true` for remote |
| `TRINO_TIMEOUT` | Query timeout (seconds) | `120` |

## Verification

All artifacts are signed with Cosign (keyless). Verify with:
```bash
cosign verify-blob --bundle mcp-trino_0.1.1_linux_amd64.tar.gz.sigstore.json \
  mcp-trino_0.1.1_linux_amd64.tar.gz
```

SLSA provenance verification:
```bash
slsa-verifier verify-artifact mcp-trino_0.1.1_linux_amd64.tar.gz \
  --provenance-path multiple.intoto.jsonl \
  --source-uri github.com/txn2/mcp-trino
```
