---
hide:
  - toc
---

# mcp-trino

**The only composable MCP server for Trino**

Connect your AI assistant to Trino data warehouses. Use it standalone or import the Go library to build custom MCP servers.

[Get Started](server/installation.md){ .md-button .md-button--primary }
[View on GitHub](https://github.com/txn2/mcp-trino){ .md-button }

---

## Two Ways to Use

<div class="grid cards" markdown>

-   :material-server:{ .lg .middle } **Use the Server**

    ---

    Connect Claude, Cursor, or any MCP client to Trino with secure defaults.

    - Read-only mode by default
    - Query limits & timeouts
    - Multi-cluster support

    [:octicons-arrow-right-24: Install in 5 minutes](server/installation.md)

-   :material-code-braces:{ .lg .middle } **Build Custom MCP**

    ---

    Import the Go library for enterprise servers with auth, tenancy, and compliance.

    - OAuth, API keys, SSO
    - Row-level tenant isolation
    - SOC2 / HIPAA audit logs

    [:octicons-arrow-right-24: View library docs](library/index.md)

</div>

---

## Why mcp-trino?

| Feature | mcp-trino | Others |
|---------|-----------|--------|
| Composable library | :material-check: | :material-close: |
| Multi-server support | :material-check: | :material-close: |
| Middleware/interceptors | :material-check: | :material-close: |
| SLSA Level 3 provenance | :material-check: | :material-close: |

---

## Available Tools

| Tool | Description |
|------|-------------|
| `trino_query` | Execute SQL with limits and format options |
| `trino_explain` | Analyze execution plans |
| `trino_list_catalogs` | Discover available catalogs |
| `trino_list_schemas` | Browse schemas in a catalog |
| `trino_list_tables` | Find tables with pattern matching |
| `trino_describe_table` | Get columns and sample data |
| `trino_list_connections` | See configured connections |

---

## Works With

Claude Desktop · Claude Code · Cursor · Windsurf · Any MCP Client
