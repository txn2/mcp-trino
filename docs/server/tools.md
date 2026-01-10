# Tools

The MCP server provides these tools for querying and exploring Trino.

## Tool Summary

| Tool | Description |
|------|-------------|
| `trino_query` | Execute SQL queries |
| `trino_explain` | Analyze execution plans |
| `trino_list_catalogs` | List available catalogs |
| `trino_list_schemas` | List schemas in a catalog |
| `trino_list_tables` | List tables in a schema |
| `trino_describe_table` | Get table columns and sample data |
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

## trino_list_catalogs

List all available data catalogs.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `connection` | string | No | default | Server connection |

### Example

> "What databases are available?"

Response:
```json
{
  "catalogs": ["hive", "iceberg", "memory", "system"]
}
```

---

## trino_list_schemas

List schemas within a catalog.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | Yes | - | Catalog name |
| `connection` | string | No | default | Server connection |

### Example

> "Show me the schemas in hive"

Response:
```json
{
  "schemas": ["default", "sales", "marketing", "analytics"]
}
```

---

## trino_list_tables

List tables in a schema with optional pattern filtering.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | Yes | - | Catalog name |
| `schema` | string | Yes | - | Schema name |
| `pattern` | string | No | - | LIKE pattern |
| `connection` | string | No | default | Server connection |

### Pattern Syntax

| Pattern | Matches |
|---------|---------|
| `order%` | Tables starting with "order" |
| `%log` | Tables ending with "log" |
| `%event%` | Tables containing "event" |

### Example

> "Show me tables related to orders"

Uses `pattern: "%order%"`:
```json
{
  "tables": ["orders", "order_items", "order_history"]
}
```

---

## trino_describe_table

Get detailed information about a table including columns and sample data.

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

1. `trino_list_catalogs` - See available databases
2. `trino_list_schemas` - Browse catalog
3. `trino_list_tables` - Find relevant tables
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
