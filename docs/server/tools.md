# Tools

The MCP server provides these tools for querying and exploring Trino.

## Tool Summary

| Tool | Description |
|------|-------------|
| `trino_query` | Execute SQL queries |
| `trino_explain` | Analyze execution plans |
| `trino_browse` | Browse catalog hierarchy (catalogs → schemas → tables) |
| `trino_describe_table` | Get columns, sample data, and semantic context |
| `trino_list_connections` | List configured server connections |

---

## trino_query

Execute SQL queries against Trino.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sql` | string | Yes | - | SQL query to execute |
| `limit` | integer | No | 1000 | Max rows (1-10000) |
| `format` | string | No | `json` | Output: `json`, `csv`, `markdown` |
| `timeout_seconds` | integer | No | 120 | Timeout (1-300) |
| `connection` | string | No | default | Server connection |

### Examples

> "Show me the first 10 customers"

```sql
SELECT * FROM customers LIMIT 10
```

> "Count orders by status"

```sql
SELECT status, COUNT(*) as count
FROM orders
GROUP BY status
```

> "Export users as CSV"

Uses `format: "csv"` parameter.

---

## trino_explain

Analyze query execution plans without running the query.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sql` | string | Yes | - | SQL query to explain |
| `type` | string | No | `LOGICAL` | Plan type |
| `connection` | string | No | default | Server connection |

### Explain Types

| Type | Use Case |
|------|----------|
| `LOGICAL` | Understand query structure |
| `DISTRIBUTED` | See execution stages across nodes |
| `IO` | Estimate data scanned |
| `VALIDATE` | Check syntax without planning |

### Examples

> "Why is this query slow?"

Uses `type: "DISTRIBUTED"` to see where time is spent.

> "How much data will this read?"

Uses `type: "IO"` to see estimated bytes scanned.

---

## trino_browse

Browse the Trino catalog hierarchy. The browsing level is determined by which parameters are provided.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | No | - | Catalog name. Omit to list all catalogs. |
| `schema` | string | No | - | Schema name. Requires catalog. Omit to list schemas. |
| `pattern` | string | No | - | LIKE pattern to filter tables (only when listing tables) |
| `connection` | string | No | default | Server connection |

### Modes

| Parameters Provided | Action |
|---------------------|--------|
| *(none)* | List all catalogs |
| `catalog` | List schemas in that catalog |
| `catalog` + `schema` | List tables in that schema |
| `catalog` + `schema` + `pattern` | List tables matching the pattern |

### Pattern Syntax

| Pattern | Matches |
|---------|---------|
| `order%` | Tables starting with "order" |
| `%log` | Tables ending with "log" |
| `%event%` | Tables containing "event" |

### Examples

> "What databases are available?"

```json
{
  "level": "catalogs",
  "items": ["hive", "iceberg", "memory", "system"],
  "count": 4
}
```

> "Show me tables related to orders"

Uses `catalog: "hive"`, `schema: "default"`, `pattern: "%order%"`:
```json
{
  "level": "tables",
  "catalog": "hive",
  "schema": "default",
  "items": ["orders", "order_items", "order_history"],
  "count": 3,
  "pattern": "%order%"
}
```

---

## trino_describe_table

Get detailed information about a table including columns, sample data, and semantic context from metadata catalogs.

### Semantic Context

When a [semantic provider](../semantic/index.md) is configured, this tool enriches the output with:

- **Description**: Business-friendly explanation of the table and columns
- **Ownership**: Data stewards and technical owners
- **Tags & Domain**: Classification labels and business domain
- **Sensitivity**: Columns marked as containing PII or sensitive data
- **Deprecation**: Warnings if the table is deprecated

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | Yes | - | Catalog name |
| `schema` | string | Yes | - | Schema name |
| `table` | string | Yes | - | Table name |
| `include_sample` | boolean | No | true | Include sample rows |
| `connection` | string | No | default | Server connection |

### Example

> "Describe the customers table"

Response:
```json
{
  "table": "hive.sales.customers",
  "columns": [
    {"name": "id", "type": "bigint", "nullable": false},
    {"name": "name", "type": "varchar(255)", "nullable": true},
    {"name": "email", "type": "varchar(255)", "nullable": true}
  ],
  "sample": [
    [1, "Alice Smith", "alice@example.com"],
    [2, "Bob Jones", "bob@example.com"]
  ]
}
```

---

## trino_list_connections

List all configured Trino server connections.

### Parameters

None.

### Example

> "What Trino servers are configured?"

Response:
```json
{
  "connections": [
    {"name": "default", "host": "prod.trino.example.com", "catalog": "hive"},
    {"name": "staging", "host": "staging.trino.example.com", "catalog": "hive"},
    {"name": "dev", "host": "localhost", "port": 8080, "ssl": false}
  ],
  "default": "default"
}
```

---

## Common Workflows

### Data Exploration

1. `trino_browse` - See available catalogs
2. `trino_browse` (with catalog) - Browse schemas
3. `trino_browse` (with catalog + schema) - Find relevant tables
4. `trino_describe_table` - Understand structure
5. `trino_query` - Query the data

### Query Optimization

1. Write initial query
2. `trino_explain` with `IO` - Check data scanned
3. Add filters to reduce data
4. `trino_explain` with `DISTRIBUTED` - Check stages
5. `trino_query` - Run optimized query

## Next Steps

- [Multi-Server](multi-server.md) - Query multiple clusters
- [Tools API Reference](../reference/tools-api.md) - Complete parameter details
