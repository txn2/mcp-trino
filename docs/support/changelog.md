# Changelog

All releases are published on GitHub with full release notes.

[:octicons-tag-24: View all releases on GitHub](https://github.com/txn2/mcp-trino/releases){ .md-button .md-button--primary }

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
