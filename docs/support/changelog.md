# Changelog

All notable changes to mcp-trino.

## Releases

| Version | Date | Highlights |
|---------|------|------------|
| v0.4.0 | 2026-02-15 | Tool description override API, enhanced AI-agent tool descriptions |
| v0.3.0 | 2026-01-30 | QueryStats improvements, CI updates |
| v0.2.1 | 2026-01-16 | SQL identifier quoting bug fix |
| v0.2.0 | 2026-01-11 | Semantic layer with DataHub and static providers |
| v0.1.1 | 2026-01-10 | Initial release |

## v0.4.0 - Tool Description Override API

**Features:**

- Tool description override API for customizing tool descriptions at runtime
- Enhanced tool descriptions with AI-agent decision context (#30, #32)
- MCP SDK bump to latest version (#28)
- CI dependency updates (#27, #29)

## v0.3.0 - QueryStats Improvements

**Features:**

- Added query ID to `QueryResult` for traceability (#22, #23, #24)
- Refactored `QueryStats` duration handling to use `DurationMs` (#26)
- Added `queryProgressUpdater` with concurrent access handling

**Maintenance:**

- CI dependency updates (#16, #19, #21)
- Repository housekeeping (#20)

## v0.2.1 - SQL Identifier Quoting Fix

**Bug Fixes:**

- Fixed SQL identifier quoting bug that caused errors with certain catalog/schema/table names (#17)

## v0.2.0 - Semantic Layer

**Features:**

- Semantic provider framework for enriching tool results with business context (#14)
- DataHub provider for surfacing descriptions, ownership, and data quality from DataHub
- Static provider for file-based semantic metadata
- Custom provider interface for building your own semantic sources
- Semantic layer documentation (#15)

## v0.1.1 - Initial Release

**Features:**

- Complete MCP server for Trino
- 7 MCP tools: query, explain, list_catalogs, list_schemas, list_tables, describe_table, list_connections
- Composable Go library for building custom MCP servers
- Multi-server support for connecting multiple Trino clusters
- Extensibility framework: middleware, interceptors, transformers
- Built-in extensions: read-only mode, logging, metrics, query logging, metadata
- File-based configuration with environment variable expansion
- Docker support

**Security:**

- Read-only mode enabled by default
- Query limits and timeouts
- SSL/TLS with certificate verification
- SLSA Level 3 provenance
- Cosign keyless signatures

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for backwards-compatible functionality additions
- **PATCH** version for backwards-compatible bug fixes

## Upgrade Guide

### Checking Current Version

```bash
mcp-trino --version
```

### Upgrading

**Homebrew:**

```bash
brew upgrade txn2/tap/mcp-trino
```

**Docker:**

```bash
docker pull ghcr.io/txn2/mcp-trino:latest
```

**Go Install:**

```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@latest
```

**Binary:**

Download the latest release from [GitHub Releases](https://github.com/txn2/mcp-trino/releases).

### Breaking Changes

Major version upgrades may include breaking changes. Check the release notes before upgrading.

## Contributing

See the [GitHub repository](https://github.com/txn2/mcp-trino) for contribution guidelines.
