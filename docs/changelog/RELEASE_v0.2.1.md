# v0.2.1 - SQL Identifier Quoting Fix

A patch release that fixes SQL identifier quoting to properly handle special characters and prevent potential injection.

## Bug Fixes

### SQL Identifier Quoting (#17)
- **Fixed**: SQL identifiers (catalog, schema, table names) are now properly quoted using double quotes per SQL standard
- **Security**: Prevents potential issues when identifiers contain special characters
- **Compatibility**: Handles reserved keywords, spaces, and special characters in identifier names

#### Affected Operations
- `SHOW SCHEMAS FROM <catalog>`
- `SHOW TABLES FROM <catalog>.<schema>`
- `DESCRIBE <catalog>.<schema>.<table>`
- `SELECT * FROM <catalog>.<schema>.<table>` (sample data queries)

#### Technical Details
Added `QuoteIdentifier()` function in `pkg/client/client.go` that:
- Wraps identifiers in double quotes
- Escapes internal double quotes by doubling them (SQL standard)
- Applied to all dynamic SQL identifier interpolation

## Installation

### Homebrew (macOS)
```bash
brew upgrade txn2/tap/mcp-trino
```

### Claude Desktop
Download the `.mcpb` bundle for your platform from the [releases page](https://github.com/txn2/mcp-trino/releases/tag/v0.2.1):
- macOS Apple Silicon: `mcp-trino_0.2.1_darwin_arm64.mcpb`
- macOS Intel: `mcp-trino_0.2.1_darwin_amd64.mcpb`
- Windows: `mcp-trino_0.2.1_windows_amd64.mcpb`

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
docker pull ghcr.io/txn2/mcp-trino:v0.2.1
```

### Go Install
```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@v0.2.1
```

## Verification

All artifacts are signed with Cosign (keyless). Verify with:
```bash
cosign verify-blob --bundle mcp-trino_0.2.1_linux_amd64.tar.gz.sigstore.json \
  mcp-trino_0.2.1_linux_amd64.tar.gz
```

## Upgrade Notes

This is a recommended upgrade for all users. No configuration changes required.
