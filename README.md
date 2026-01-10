# mcp-trino

A Model Context Protocol (MCP) server for [Trino](https://trino.io/), enabling AI assistants like Claude to query and explore data warehouses.

[![GitHub license](https://img.shields.io/github/license/txn2/mcp-trino.svg)](https://github.com/txn2/mcp-trino/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/txn2/mcp-trino.svg)](https://pkg.go.dev/github.com/txn2/mcp-trino)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/mcp-trino)](https://goreportcard.com/report/github.com/txn2/mcp-trino)
[![codecov](https://codecov.io/gh/txn2/mcp-trino/branch/main/graph/badge.svg)](https://codecov.io/gh/txn2/mcp-trino)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/txn2/mcp-trino/badge)](https://scorecard.dev/viewer/?uri=github.com/txn2/mcp-trino)
[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)

This project provides both a **standalone MCP server** and a **composable Go library**. The standalone server serves as a reference implementation, demonstrating all extensibility features including middleware, query interceptors, and result transformers. Import the `pkg/tools` and `pkg/extensions` packages into your own MCP server to add Trino capabilities with full customization for authentication, tenant isolation, audit logging, and more.

## Features

- **SQL Query Execution**: Execute queries with configurable limits and timeouts
- **Execution Plans**: Analyze query plans (logical, distributed, IO)
- **Schema Discovery**: List catalogs, schemas, and tables
- **Table Inspection**: Describe columns and sample data
- **Composable Design**: Import as a Go package to build custom MCP servers

## Installation

### As a standalone server

```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@latest
```

### As a library

```bash
go get github.com/txn2/mcp-trino
```

## Quick Start

### Standalone Server

```bash
# Set environment variables
export TRINO_HOST=trino.example.com
export TRINO_USER=your_user
export TRINO_PASSWORD=your_password
export TRINO_CATALOG=hive
export TRINO_SCHEMA=default

# Run the server
mcp-trino
```

### Claude Code CLI

1. **Clone and build the binary:**

```bash
git clone https://github.com/txn2/mcp-trino.git ~/mcp-trino
cd ~/mcp-trino
go build -o mcp-trino ./cmd/mcp-trino
```

2. **Add the MCP server to Claude Code:**

```bash
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_PORT=443 \
  -e TRINO_SSL=true \
  -e TRINO_USER=your_username \
  -e TRINO_PASSWORD=your_password \
  -e TRINO_CATALOG=hive \
  -e TRINO_SCHEMA=default \
  -- ~/mcp-trino/mcp-trino
```

3. **Restart Claude Code** to load the new MCP server.

4. **Verify the tools are available** by asking Claude to list your Trino catalogs.

**Multiple Trino Servers:**

You can add multiple Trino instances with different names:

```bash
# Production
claude mcp add trino-prod \
  -e TRINO_HOST=trino.prod.example.com \
  -e TRINO_USER=prod_user \
  -e TRINO_PASSWORD=prod_pass \
  -- ~/mcp-trino/mcp-trino

# Staging
claude mcp add trino-staging \
  -e TRINO_HOST=trino.staging.example.com \
  -e TRINO_USER=staging_user \
  -e TRINO_PASSWORD=staging_pass \
  -- ~/mcp-trino/mcp-trino
```

Each server gets its own set of tools (e.g., `trino-prod_query`, `trino-staging_query`).

**Update or remove a configuration:**

```bash
claude mcp remove trino
claude mcp add trino -e TRINO_HOST=... -- ~/mcp-trino/mcp-trino
```

### Claude Desktop Configuration

Add to your Claude Desktop `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "trino": {
      "command": "/Users/you/mcp-trino/mcp-trino",
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

## Tools

| Tool | Description |
|------|-------------|
| `trino_query` | Execute SQL queries with limit/timeout control |
| `trino_explain` | Get execution plans (logical/distributed/io/validate) |
| `trino_list_catalogs` | List available catalogs |
| `trino_list_schemas` | List schemas in a catalog |
| `trino_list_tables` | List tables in a schema |
| `trino_describe_table` | Get table columns and sample data |

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
