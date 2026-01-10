# MCP Server

The mcp-trino server enables AI assistants to query Trino data warehouses through the Model Context Protocol (MCP).

## What It Does

When connected to your AI assistant, the server provides tools for:

- **SQL Execution** - Run queries with automatic limits and timeouts
- **Query Analysis** - Explain execution plans before running expensive queries
- **Schema Exploration** - Browse catalogs, schemas, tables, and columns
- **Multi-Cluster** - Query multiple Trino clusters from one installation

## Supported AI Clients

| Client | Platform | Install Method |
|--------|----------|----------------|
| Claude Desktop | macOS, Windows | `.mcpb` bundle |
| Claude Code | CLI | `claude mcp add` |
| Cursor | macOS, Windows, Linux | Manual config |
| Windsurf | macOS, Windows, Linux | Manual config |
| Any MCP Client | Any | stdio transport |

## Quick Demo

Once installed, ask your AI assistant:

> "What tables are in the sales schema?"

The assistant uses `trino_list_tables` to show available tables.

> "Show me the top 10 customers by order value"

The assistant generates SQL and uses `trino_query` to execute it.

> "Why is my orders query slow?"

The assistant uses `trino_explain` to analyze the execution plan.

## Features

### Secure Defaults

- **Read-only mode** enabled by default (blocks INSERT, UPDATE, DELETE)
- **Row limits** prevent excessive data retrieval (default: 1000, max: 10000)
- **Query timeouts** prevent runaway queries (default: 120s)

### Production Ready

- File-based configuration for Kubernetes ConfigMaps
- Environment variable expansion for secrets
- Multi-server support for environment separation

### Enterprise Security

- SLSA Level 3 provenance on all releases
- Cosign keyless signatures for verification
- Checksum verification

## Next Steps

- [Installation](installation.md) - Install the server
- [Configuration](configuration.md) - Configure connection and extensions
- [Tools](tools.md) - Learn about available tools
- [Multi-Server](multi-server.md) - Connect multiple clusters
