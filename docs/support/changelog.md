# Changelog

All notable changes to mcp-trino.

## Releases

| Version | Date | Highlights |
|---------|------|------------|
| v0.1.1 | 2025-01-10 | Initial release |

## v0.1.1 - Initial Release

**Features:**

- Complete MCP server for Trino
- 7 MCP tools: query, explain, list_catalogs, list_schemas, list_tables, describe_table, list_connections
- Composable Go library for building custom MCP servers
- Multi-server support for connecting multiple Trino clusters
- Extensibility framework: middleware, interceptors, transformers
- Built-in extensions: read-only mode, logging, metrics, query logging, metadata
- File-based configuration with environment variable expansion
- Docker and Kubernetes support

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
