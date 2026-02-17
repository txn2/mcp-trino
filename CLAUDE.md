# CLAUDE.md

This file provides guidance to Claude Code when working with this project.

## Project Overview

**mcp-trino** is a generic, open-source MCP (Model Context Protocol) server for Trino. It enables AI assistants to query and explore data warehouses via the MCP protocol.

**Key Design Goals:**
- Composable: Can be used standalone OR imported as a library
- Generic: No domain-specific logic; suitable for any Trino deployment
- Secure: Configurable limits and timeouts to prevent abuse

## Code Standards

1. **Idiomatic Go**: All code must follow idiomatic Go patterns and conventions. Use `gofmt`, follow Effective Go guidelines, and adhere to Go Code Review Comments.

2. **Test Coverage**: Project must maintain >80% unit test coverage. Build mocks where necessary to achieve this. Use table-driven tests where appropriate.

3. **Testing Definition**: When asked to "test" or "testing" the code, this means running the full CI test suite:
   - Unit tests with race detection (`make test` or `go test -race ./...`)
   - Linting (`make lint` / golangci-lint)
   - Security scanning (`gosec ./...`)
   - Cyclomatic complexity (`gocyclo -over 15 .` must return no results)
   - All checks that run in CI must pass locally before considering code "tested"

4. **Human Review Required**: A human must review and approve every line of code before it is committed. Therefore, commits are always performed by a human, not by Claude.

5. **Go Report Card**: The project MUST always maintain 100% across all categories on [Go Report Card](https://goreportcard.com/).

## Project Structure

```
mcp-trino/
├── cmd/mcp-trino/main.go      # Standalone server entrypoint
├── pkg/                        # PUBLIC API (importable by other projects)
│   ├── client/                 # Trino client wrapper
│   │   ├── client.go           # Connection and query execution
│   │   └── config.go           # Configuration from env/struct
│   └── tools/                  # MCP tool definitions
│       ├── toolkit.go          # NewToolkit() and RegisterAll()
│       ├── query.go            # trino_query tool
│       ├── explain.go          # trino_explain tool
│       └── schema.go           # list_catalogs, list_schemas, etc.
├── internal/server/            # Default server setup (private)
├── go.mod
├── LICENSE                     # Apache 2.0
└── README.md
```

## Key Dependencies

- `github.com/modelcontextprotocol/go-sdk` - Official MCP SDK (v1.0.0)
- `github.com/trinodb/trino-go-client` - Trino Go driver

## Building and Running

```bash
# Build
go build -o mcp-trino ./cmd/mcp-trino

# Run (requires Trino connection)
export TRINO_HOST=trino.example.com
export TRINO_USER=user
export TRINO_PASSWORD=pass
export TRINO_CATALOG=hive
export TRINO_SCHEMA=default
./mcp-trino
```

## Testing with Docker

```bash
# Start local Trino
docker run -d -p 8080:8080 --name trino trinodb/trino

# Configure for local testing
export TRINO_HOST=localhost
export TRINO_PORT=8080
export TRINO_USER=admin
export TRINO_SSL=false
export TRINO_CATALOG=memory
export TRINO_SCHEMA=default

./mcp-trino
```

## Composition Pattern

This package is designed to be imported by other MCP servers:

```go
import (
    "github.com/txn2/mcp-trino/pkg/client"
    "github.com/txn2/mcp-trino/pkg/tools"
)

// Create client
trinoClient, _ := client.New(client.FromEnv())

// Create toolkit and register on your server
toolkit := tools.NewToolkit(trinoClient, tools.DefaultConfig())
toolkit.RegisterAll(yourMCPServer)
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `trino_query` | Execute read-only SQL (SELECT/SHOW/DESCRIBE), returns JSON/CSV/markdown |
| `trino_execute` | Execute any SQL including write ops (INSERT/UPDATE/DELETE/CREATE/DROP) |
| `trino_explain` | Get execution plan |
| `trino_list_catalogs` | List available catalogs |
| `trino_list_schemas` | List schemas in catalog |
| `trino_list_tables` | List tables in schema |
| `trino_describe_table` | Get column definitions |

## Configuration Reference

Environment variables:
- `TRINO_HOST` - Server hostname
- `TRINO_PORT` - Server port
- `TRINO_USER` - Username (required)
- `TRINO_PASSWORD` - Password
- `TRINO_CATALOG` - Default catalog
- `TRINO_SCHEMA` - Default schema
- `TRINO_SSL` - Enable SSL (true/false)
- `TRINO_SSL_VERIFY` - Verify SSL certs
- `TRINO_TIMEOUT` - Query timeout in seconds
- `TRINO_SOURCE` - Client source identifier
