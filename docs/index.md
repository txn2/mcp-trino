---
description: An MCP server that connects AI assistants to Trino data warehouses
hide:
  - navigation
---

# txn2/mcp-trino

An MCP server that connects AI assistants to Trino data warehouses. Execute SQL queries, explore schemas, and describe tables with optional semantic context from metadata catalogs like DataHub.

Unlike other MCP servers, mcp-trino is designed as a composable Go library. Import it into your own MCP server to add Trino capabilities with custom authentication, tenant isolation, and audit logging. The standalone server works out of the box; the library lets you build exactly what your organization needs.

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

## Core Capabilities

<div class="grid cards" markdown>

-   :material-puzzle:{ .lg .middle } **Composable Architecture**

    ---

    Import as a Go library to build custom MCP servers with authentication,
    tenant isolation, and audit logging without forking.

    [:octicons-arrow-right-24: Library docs](library/index.md)

-   :material-database-search:{ .lg .middle } **Semantic Context**

    ---

    Surface business descriptions, ownership, data quality, and sensitivity
    markers from metadata catalogs like DataHub.

    [:octicons-arrow-right-24: Semantic layer](semantic/index.md)

-   :material-server-network:{ .lg .middle } **Multi-Cluster**

    ---

    Query production, staging, and development Trino servers from a single
    MCP installation with unified credentials.

    [:octicons-arrow-right-24: Multi-server setup](server/multi-server.md)

-   :material-shield-check:{ .lg .middle } **Secure Defaults**

    ---

    Read-only mode, query limits, timeouts, and SLSA Level 3 provenance
    for production deployments.

    [:octicons-arrow-right-24: Security reference](reference/security.md)

</div>

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
