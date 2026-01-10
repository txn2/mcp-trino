# Tools API Reference

Complete parameter and response specifications for all mcp-trino tools.

---

## trino_query

Execute SQL queries against Trino.

### Parameters

| Parameter | Type | Required | Default | Constraints | Description |
|-----------|------|----------|---------|-------------|-------------|
| `sql` | string | **Yes** | - | Non-empty | SQL query to execute |
| `limit` | integer | No | 1000 | 1-10000 | Maximum rows to return |
| `format` | string | No | `json` | `json`, `csv`, `markdown` | Output format |
| `timeout_seconds` | integer | No | 120 | 1-300 | Query timeout |
| `connection` | string | No | `default` | Valid connection name | Server connection |

### Response

**JSON Format:**

```json
{
  "columns": ["id", "name", "created_at"],
  "rows": [
    [1, "Alice", "2024-01-15T10:00:00Z"],
    [2, "Bob", "2024-01-15T11:00:00Z"]
  ],
  "row_count": 2,
  "truncated": false
}
```

**CSV Format:**

```csv
id,name,created_at
1,Alice,2024-01-15T10:00:00Z
2,Bob,2024-01-15T11:00:00Z
```

**Markdown Format:**

```markdown
| id | name | created_at |
|----|------|------------|
| 1 | Alice | 2024-01-15T10:00:00Z |
| 2 | Bob | 2024-01-15T11:00:00Z |
```

### Errors

| Code | Message | Cause |
|------|---------|-------|
| `INVALID_SQL` | SQL query is required | Empty `sql` parameter |
| `LIMIT_EXCEEDED` | Limit exceeds maximum | `limit` > 10000 |
| `TIMEOUT_EXCEEDED` | Timeout exceeds maximum | `timeout_seconds` > 300 |
| `QUERY_ERROR` | Query execution failed | Trino error |
| `READ_ONLY` | Write operation blocked | INSERT/UPDATE/DELETE in read-only mode |

---

## trino_explain

Get query execution plan without running the query.

### Parameters

| Parameter | Type | Required | Default | Constraints | Description |
|-----------|------|----------|---------|-------------|-------------|
| `sql` | string | **Yes** | - | Non-empty | SQL query to explain |
| `type` | string | No | `LOGICAL` | See below | Explain type |
| `connection` | string | No | `default` | Valid connection name | Server connection |

### Explain Types

| Type | Description | Use Case |
|------|-------------|----------|
| `LOGICAL` | Logical query plan | Understand query structure |
| `DISTRIBUTED` | Physical execution plan | See worker distribution |
| `IO` | I/O statistics estimate | Estimate data scanned |
| `VALIDATE` | Syntax validation only | Check without planning |

### Response

```json
{
  "type": "LOGICAL",
  "plan": "- Output[columnNames = [id, name]] => [[id, name]]\n    - TableScan[table = hive:default:users] => [[id, name]]"
}
```

### Errors

| Code | Message | Cause |
|------|---------|-------|
| `INVALID_SQL` | SQL query is required | Empty `sql` parameter |
| `INVALID_TYPE` | Invalid explain type | Unknown type value |
| `PLAN_ERROR` | Failed to generate plan | Invalid SQL syntax |

---

## trino_list_catalogs

List all available data catalogs.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `connection` | string | No | `default` | Server connection |

### Response

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
| `catalog` | string | **Yes** | - | Catalog name |
| `connection` | string | No | `default` | Server connection |

### Response

```json
{
  "catalog": "hive",
  "schemas": ["default", "sales", "marketing", "analytics"]
}
```

### Errors

| Code | Message | Cause |
|------|---------|-------|
| `CATALOG_REQUIRED` | Catalog is required | Empty `catalog` |
| `CATALOG_NOT_FOUND` | Catalog not found | Invalid catalog name |

---

## trino_list_tables

List tables in a schema with optional filtering.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | **Yes** | - | Catalog name |
| `schema` | string | **Yes** | - | Schema name |
| `pattern` | string | No | - | LIKE pattern filter |
| `connection` | string | No | `default` | Server connection |

### Pattern Syntax

| Pattern | Matches |
|---------|---------|
| `order%` | Tables starting with "order" |
| `%log` | Tables ending with "log" |
| `%event%` | Tables containing "event" |
| `user_` | Tables like "user_1", "user_a" |

### Response

```json
{
  "catalog": "hive",
  "schema": "sales",
  "tables": ["customers", "orders", "order_items"],
  "pattern": null
}
```

### Errors

| Code | Message | Cause |
|------|---------|-------|
| `CATALOG_REQUIRED` | Catalog is required | Empty `catalog` |
| `SCHEMA_REQUIRED` | Schema is required | Empty `schema` |
| `SCHEMA_NOT_FOUND` | Schema not found | Invalid schema name |

---

## trino_describe_table

Get table structure and optional sample data.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | **Yes** | - | Catalog name |
| `schema` | string | **Yes** | - | Schema name |
| `table` | string | **Yes** | - | Table name |
| `include_sample` | boolean | No | `true` | Include sample rows |
| `connection` | string | No | `default` | Server connection |

### Response

```json
{
  "table": "hive.sales.customers",
  "columns": [
    {
      "name": "id",
      "type": "bigint",
      "nullable": false,
      "comment": "Primary key"
    },
    {
      "name": "name",
      "type": "varchar(255)",
      "nullable": true,
      "comment": null
    },
    {
      "name": "email",
      "type": "varchar(255)",
      "nullable": true,
      "comment": "Contact email"
    }
  ],
  "sample": [
    [1, "Alice Smith", "alice@example.com"],
    [2, "Bob Jones", "bob@example.com"]
  ],
  "sample_count": 2
}
```

### Errors

| Code | Message | Cause |
|------|---------|-------|
| `CATALOG_REQUIRED` | Catalog is required | Empty `catalog` |
| `SCHEMA_REQUIRED` | Schema is required | Empty `schema` |
| `TABLE_REQUIRED` | Table is required | Empty `table` |
| `TABLE_NOT_FOUND` | Table not found | Invalid table name |

---

## trino_list_connections

List all configured server connections.

### Parameters

None.

### Response

```json
{
  "connections": [
    {
      "name": "default",
      "host": "prod.trino.example.com",
      "port": 443,
      "catalog": "hive",
      "ssl": true
    },
    {
      "name": "staging",
      "host": "staging.trino.example.com",
      "port": 443,
      "catalog": "hive",
      "ssl": true
    },
    {
      "name": "dev",
      "host": "localhost",
      "port": 8080,
      "catalog": "memory",
      "ssl": false
    }
  ],
  "default": "default"
}
```

---

## Common Parameters

### connection

All tools accept an optional `connection` parameter to specify which Trino server to use:

```json
{
  "tool": "trino_query",
  "arguments": {
    "sql": "SELECT * FROM users",
    "connection": "staging"
  }
}
```

If not specified, the default connection is used.

### Error Response Format

All errors follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "sql": "SELECT * FROM nonexistent",
      "trino_error_code": 1
    }
  }
}
```

---

## Rate Limits

The toolkit enforces these default limits:

| Limit | Default | Maximum | Environment Variable |
|-------|---------|---------|---------------------|
| Rows per query | 1000 | 10000 | - |
| Query timeout | 120s | 300s | `TRINO_TIMEOUT` |

Override in toolkit configuration:

```go
cfg := tools.Config{
    DefaultLimit:   500,
    MaxLimit:       5000,
    DefaultTimeout: 60 * time.Second,
    MaxTimeout:     180 * time.Second,
}
```
