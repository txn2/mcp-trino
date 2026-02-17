![txn2/mcp-trino](./docs/images/txn2_mcp_trino_banner.png)

[![GitHub license](https://img.shields.io/github/license/txn2/mcp-trino.svg)](https://github.com/txn2/mcp-trino/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/txn2/mcp-trino.svg)](https://pkg.go.dev/github.com/txn2/mcp-trino)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/mcp-trino)](https://goreportcard.com/report/github.com/txn2/mcp-trino)
[![codecov](https://codecov.io/gh/txn2/mcp-trino/branch/main/graph/badge.svg)](https://codecov.io/gh/txn2/mcp-trino)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/txn2/mcp-trino/badge)](https://scorecard.dev/viewer/?uri=github.com/txn2/mcp-trino)
[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)

**Full documentation at [mcp-trino.txn2.com](https://mcp-trino.txn2.com)**

A Model Context Protocol (MCP) server for [Trino](https://trino.io/), enabling AI assistants to query and explore data warehouses with optional semantic context from metadata catalogs.

AI assistants excel at querying data but lack organizational context: which tables are trustworthy, what metrics mean, and which columns contain sensitive data. mcp-trino bridges this gap by connecting Trino to AI assistants through the MCP protocol, with an optional semantic layer that surfaces business metadata alongside query results.

## Core Capabilities

**Composable Architecture**
- Import as a Go library to build custom MCP servers
- Add authentication, tenant isolation, audit logging without forking
- Middleware and interceptor patterns for enterprise requirements

**Semantic Context**
- Surface business descriptions, ownership, and data quality from metadata catalogs
- Mark sensitive columns and deprecation warnings for AI assistants
- Connect to DataHub, static files, or build custom metadata providers

**Multi-Cluster Connectivity**
- Query multiple Trino servers from a single MCP installation
- Unified interface across production, staging, and development environments

**Secure Defaults**
- Read-only mode prevents accidental data modification
- Query limits and timeouts prevent runaway operations
- SLSA Level 3 provenance for supply chain security

## Features

- **Execute SQL Queries**: Run queries with configurable row limits and timeouts
- **Analyze Execution Plans**: Inspect logical, distributed, and I/O query plans
- **Discover Schema**: Browse catalogs, schemas, and tables across clusters
- **Describe Tables**: View column definitions with optional data samples
- **Enrich with Context**: Surface business metadata, ownership, and data quality
- **Compose Custom Servers**: Import as a Go library with middleware and interceptors

## Installation

### Homebrew (macOS)

```bash
brew install txn2/tap/mcp-trino
```

### Claude Desktop

Claude Desktop is the GUI application for chatting with Claude. Install the mcp-trino extension to enable Trino queries in your conversations.

**Option 1: One-Click Install (Recommended)**

Download the `.mcpb` bundle for your Mac from the [releases page](https://github.com/txn2/mcp-trino/releases) and double-click to install:

| Mac Type | Chip | Download |
|----------|------|----------|
| MacBook Air/Pro (2020+), Mac Mini (2020+), iMac (2021+), Mac Studio | Apple M1, M2, M3, M4 (arm64) | `mcp-trino_*_darwin_arm64.mcpb` |
| MacBook Air/Pro (pre-2020), Mac Mini (pre-2020), iMac (pre-2021) | Intel (amd64) | `mcp-trino_*_darwin_amd64.mcpb` |

> **Tip:** Not sure which chip you have? Click  → "About This Mac". Look for "Chip" (Apple Silicon) or "Processor" (Intel).

**Option 2: Manual Configuration**

Add to your `claude_desktop_config.json` (find via Claude Desktop → Settings → Developer):

```json
{
  "mcpServers": {
    "trino": {
      "command": "/opt/homebrew/bin/mcp-trino",
      "env": {
        "TRINO_HOST": "trino.example.com",
        "TRINO_USER": "your_user",
        "TRINO_PASSWORD": "your_password",
        "TRINO_CATALOG": "hive",
        "TRINO_SCHEMA": "default"
      }
    }
  }
}
```

### Claude Code CLI

Claude Code is the terminal-based coding assistant. Add mcp-trino as an MCP server:

```bash
# Install via Homebrew first (see above), then:
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  -e TRINO_CATALOG=hive \
  -- mcp-trino
```

Or download and install manually:

```bash
# Download the latest release for your architecture
curl -L https://github.com/txn2/mcp-trino/releases/latest/download/mcp-trino_$(uname -s)_$(uname -m).tar.gz | tar xz

# Add to Claude Code
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  -e TRINO_CATALOG=hive \
  -- ./mcp-trino
```

### Docker

```bash
docker run --rm -i \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  ghcr.io/txn2/mcp-trino:latest
```

### Go Install

```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@latest
```

### Download Binary

Download pre-built binaries from the [releases page](https://github.com/txn2/mcp-trino/releases). All releases are signed with [Cosign](https://github.com/sigstore/cosign) and include [SLSA provenance](https://slsa.dev/).

### As a Library

```bash
go get github.com/txn2/mcp-trino
```

## Quick Start

### Multiple Trino Servers

You can configure multiple Trino instances with different names:

```bash
# Production
claude mcp add trino-prod \
  -e TRINO_HOST=trino.prod.example.com \
  -e TRINO_USER=prod_user \
  -- mcp-trino

# Staging
claude mcp add trino-staging \
  -e TRINO_HOST=trino.staging.example.com \
  -e TRINO_USER=staging_user \
  -- mcp-trino
```

### Standalone Server

```bash
export TRINO_HOST=trino.example.com
export TRINO_USER=your_user
export TRINO_PASSWORD=your_password
mcp-trino
```

## Tools

| Tool | Description |
|------|-------------|
| `trino_query` | Execute read-only SQL queries (SELECT, SHOW, DESCRIBE) with limit/timeout control |
| `trino_execute` | Execute any SQL including write operations (INSERT, UPDATE, DELETE, CREATE, DROP) |
| `trino_explain` | Get execution plans (logical/distributed/io/validate) |
| `trino_list_catalogs` | List available catalogs |
| `trino_list_schemas` | List schemas in a catalog |
| `trino_list_tables` | List tables in a schema |
| `trino_describe_table` | Get columns, sample data, and semantic context (if configured) |
| `trino_list_connections` | List all configured server connections |

## Semantic Layer

AI agents operate more reliably when they understand organizational context: not just table structures, but which datasets are production-ready, what business terms mean, and which columns require careful handling.

mcp-trino's semantic layer integrates with metadata catalogs to surface this context alongside query results:

| Metadata | Description |
|----------|-------------|
| **Descriptions** | Business-friendly explanations of tables and columns |
| **Ownership** | Data stewards and technical owners |
| **Tags & Domains** | Classification labels and business domains |
| **Glossary Terms** | Links to formal business definitions |
| **Data Quality** | Freshness scores and quality metrics |
| **Sensitivity** | PII and sensitive data markers at column level |
| **Deprecation** | Warnings with replacement guidance |
| **Lineage** | Upstream and downstream data dependencies |

### Providers

| Provider | Description |
|----------|-------------|
| **DataHub** | Connect to DataHub's GraphQL API for enterprise metadata |
| **Static Files** | Load metadata from YAML or JSON files with hot-reload |
| **Custom** | Implement the `semantic.Provider` interface for any catalog |

See the [Semantic Layer Documentation](https://mcp-trino.txn2.com/semantic/) for configuration, caching, and custom provider development.

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `TRINO_HOST` | Trino server hostname | `localhost` |
| `TRINO_PORT` | Trino server port | `443` (SSL) / `8080` |
| `TRINO_USER` | Authentication username | (required) |
| `TRINO_PASSWORD` | Authentication password | (optional) |
| `TRINO_CATALOG` | Default catalog | `memory` |
| `TRINO_SCHEMA` | Default schema | `default` |
| `TRINO_SSL` | Enable HTTPS | `true` for remote hosts |
| `TRINO_SSL_VERIFY` | Verify SSL certificates | `true` |
| `TRINO_TIMEOUT` | Query timeout (seconds) | `120` |
| `TRINO_SOURCE` | Client identifier | `mcp-trino` |
| `TRINO_ADDITIONAL_SERVERS` | Additional servers (JSON) | (optional) |

### Multi-Server Configuration

Connect to multiple Trino servers from a single installation. Configure your primary server with the standard environment variables, then add additional servers via JSON:

```bash
export TRINO_HOST=prod.trino.example.com
export TRINO_USER=admin
export TRINO_PASSWORD=secret
export TRINO_ADDITIONAL_SERVERS='{
  "staging": {"host": "staging.trino.example.com"},
  "dev": {"host": "localhost", "port": 8080, "ssl": false}
}'
```

Additional servers inherit credentials and settings from the primary server unless overridden:

```json
{
  "staging": {
    "host": "staging.trino.example.com",
    "user": "staging_user",
    "catalog": "iceberg"
  },
  "dev": {
    "host": "localhost",
    "port": 8080,
    "ssl": false,
    "user": "admin"
  }
}
```

Use the `connection` parameter in any tool to target a specific server:

```
"Query the staging server: SELECT * FROM users LIMIT 10"
→ trino_query(sql="...", connection="staging")
```

Use `trino_list_connections` to discover available connections.

### File-Based Configuration

For production deployments using Kubernetes ConfigMaps, Vault, or other secret management systems, mcp-trino supports file-based configuration:

```yaml
# config.yaml
trino:
  host: trino.example.com
  port: 443
  user: ${TRINO_USER}           # Supports env var expansion
  password: ${TRINO_PASSWORD}   # Secrets can come from env
  catalog: hive
  schema: default
  ssl: true
  timeout: 120s

toolkit:
  default_limit: 1000
  max_limit: 10000
  default_timeout: 120s
  max_timeout: 300s

extensions:
  logging: true
  readonly: true
  errors: true
```

Load configuration in your custom server:

```go
import "github.com/txn2/mcp-trino/pkg/extensions"

// Load from file with env var overrides
cfg, err := extensions.LoadConfig("/etc/mcp-trino/config.yaml")

// Convert to individual configs
clientCfg := cfg.ClientConfig()
toolsCfg := cfg.ToolsConfig()
extCfg := cfg.ExtConfig()
```

## Using as a Library

mcp-trino is designed to be composable. You can import its tools into your own MCP server:

```go
package main

import (
    "context"
    "log"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/txn2/mcp-trino/pkg/client"
    "github.com/txn2/mcp-trino/pkg/tools"
)

func main() {
    // Create your MCP server
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "my-data-server",
        Version: "1.0.0",
    }, nil)

    // Create Trino client
    trinoClient, err := client.New(client.Config{
        Host:    "trino.example.com",
        Port:    443,
        User:    "service_user",
        SSL:     true,
        Catalog: "hive",
        Schema:  "analytics",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer trinoClient.Close()

    // Add Trino tools to your server
    toolkit := tools.NewToolkit(trinoClient, tools.Config{
        DefaultLimit: 1000,
        MaxLimit:     10000,
    })
    toolkit.RegisterAll(server)

    // Add your own custom tools here...
    // mcp.AddTool(server, &mcp.Tool{...}, handler)

    // Run the server
    if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
        log.Fatal(err)
    }
}
```

## Extensions

The standalone server includes optional extensions that can be enabled via environment variables:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `MCP_TRINO_EXT_LOGGING` | `false` | Structured JSON logging of tool calls |
| `MCP_TRINO_EXT_METRICS` | `false` | In-memory metrics collection |
| `MCP_TRINO_EXT_READONLY` | `true` | Block modification statements (INSERT, UPDATE, DELETE, etc.) |
| `MCP_TRINO_EXT_QUERYLOG` | `false` | Log all SQL queries for audit |
| `MCP_TRINO_EXT_METADATA` | `false` | Add execution metadata footer to results |
| `MCP_TRINO_EXT_ERRORS` | `true` | Add helpful hints to error messages |

### Using Extensions in Custom Servers

```go
import (
    "github.com/txn2/mcp-trino/pkg/extensions"
    "github.com/txn2/mcp-trino/pkg/tools"
)

// Load extension config from environment
extCfg := extensions.FromEnv()

// Or configure programmatically
extCfg := extensions.Config{
    EnableLogging:   true,
    EnableReadOnly:  true,
    EnableErrorHelp: true,
}

// Build toolkit options from extensions
toolkitOpts := extensions.BuildToolkitOptions(extCfg)

// Create toolkit with extensions
toolkit := tools.NewToolkit(trinoClient, toolsCfg, toolkitOpts...)
```

### Custom Middleware and Interceptors

You can create custom middleware, interceptors, and transformers:

```go
// Custom middleware for authentication
authMiddleware := tools.MiddlewareFunc{
    BeforeFn: func(ctx context.Context, tc *tools.ToolContext) (context.Context, error) {
        // Validate user permissions
        return ctx, nil
    },
}

// Custom interceptor for tenant isolation
tenantInterceptor := tools.QueryInterceptorFunc(
    func(ctx context.Context, sql string, toolName tools.ToolName) (string, error) {
        // Add WHERE tenant_id = ? clause
        return sql, nil
    },
)

// Apply to toolkit
toolkit := tools.NewToolkit(client, cfg,
    tools.WithMiddleware(authMiddleware),
    tools.WithQueryInterceptor(tenantInterceptor),
)
```

## Security Considerations

- **Credentials**: Store passwords in environment variables or secret managers
- **Query Limits**: Default 1000 rows, max 10000 to prevent data exfiltration
- **Timeouts**: Default 120s timeout prevents runaway queries
- **Read-Only**: ReadOnly interceptor enabled by default blocks modification statements
- **Access Control**: Configure Trino roles and catalog access for defense in depth

## Development

```bash
# Clone the repository
git clone https://github.com/txn2/mcp-trino.git
cd mcp-trino

# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make verify

# Run with a local Trino (e.g., via Docker)
make docker-trino
export TRINO_HOST=localhost
export TRINO_PORT=8080
export TRINO_USER=admin
export TRINO_SSL=false
./mcp-trino
```

## Contributing

We welcome contributions for bug fixes, tests, and documentation. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[Apache License 2.0](LICENSE)

## Related Projects

- [Model Context Protocol](https://modelcontextprotocol.io/) - The MCP specification
- [Trino](https://trino.io/) - Distributed SQL query engine
- [Official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - Go SDK for MCP

---

Open source by [Craig Johnston](https://twitter.com/cjimti), sponsored by [Deasil Works, Inc.](https://deasil.works/)
