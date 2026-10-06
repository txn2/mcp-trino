# Tools API Reference

Complete parameter and response specifications for all mcp-trino tools.

## Tool Annotations

All tools declare MCP behavioral annotations that help AI agents understand side effects:

| Tool | ReadOnly | Destructive | Idempotent | OpenWorld |
|------|----------|-------------|------------|-----------|
| `trino_query` | false | **false** | false | false |
| `trino_explain` | true | — | true | false |
| `trino_browse` | true | — | true | false |
| `trino_describe_table` | true | — | true | false |
| `trino_list_connections` | true | — | true | false |

Annotations can be overridden at the toolkit or per-registration level. See [Extensibility: Tool Annotations](../library/extensibility.md#tool-annotations).

## Structured Outputs

All tools return typed structured output alongside the human-readable text response. This enables programmatic access to results without parsing text. Output types are documented in each tool's section below.

Each tool advertises an explicit JSON Schema for its structured output. The schemas are open — no top-level `required` list and no `additionalProperties: false` — so a host that composes mcp-trino and adds keys to `structuredContent` still produces results that validate. Schemas can be overridden at the toolkit or per-registration level. See [Extensibility: Advertised Output Schemas](../library/extensibility.md#advertised-output-schemas).

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

### Structured Output (`QueryOutput`)

```json
{
  "columns": [{"name": "id", "type": "bigint"}, {"name": "name", "type": "varchar"}],
  "rows": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}],
  "row_count": 2,
  "stats": {
    "row_count": 2,
    "truncated": false,
    "limit_applied": 1000,
    "duration_ms": 42
  }
}
```

A failed query's error result also carries an `error` object, alongside zero-valued result fields. See [Error Response Format](#error-response-format).

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

### Structured Output (`ExplainOutput`)

```json
{
  "plan": "- Output[columnNames = [id, name]] => ...",
  "type": "LOGICAL"
}
```

---

## trino_browse

Browse the Trino catalog hierarchy. The browsing level is determined by which parameters are provided.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `catalog` | string | No | - | Catalog name. Omit to list all catalogs. |
| `schema` | string | No | - | Schema name. Requires catalog. Omit to list schemas. |
| `pattern` | string | No | - | LIKE pattern to filter tables (only when listing tables) |
| `connection` | string | No | `default` | Server connection |

### Modes

| Parameters | Action |
|------------|--------|
| *(none)* | List all catalogs |
| `catalog` | List schemas in that catalog |
| `catalog` + `schema` | List tables in that schema |

### Pattern Syntax (tables mode)

| Pattern | Matches |
|---------|---------|
| `order%` | Tables starting with "order" |
| `%log` | Tables ending with "log" |
| `%event%` | Tables containing "event" |

### Errors

| Message | Cause |
|---------|-------|
| `schema requires catalog` | `schema` provided without `catalog` |
| `pattern requires both catalog and schema` | `pattern` provided without both `catalog` and `schema` |

### Structured Output (`BrowseOutput`)

```json
{
  "level": "tables",
  "catalog": "hive",
  "schema": "sales",
  "items": ["customers", "orders", "order_items"],
  "count": 3,
  "pattern": "%order%"
}
```

The `level` field indicates which mode was used: `"catalogs"`, `"schemas"`, or `"tables"`.

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

### Structured Output (`DescribeTableOutput`)

```json
{
  "catalog": "hive",
  "schema": "sales",
  "table": "customers",
  "columns": [
    {"name": "id", "type": "bigint", "nullable": "NO", "comment": "Primary key"},
    {"name": "name", "type": "varchar(255)", "nullable": "YES"},
    {"name": "email", "type": "varchar(255)", "nullable": "YES", "comment": "Contact email"}
  ],
  "column_count": 3
}
```

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

### Structured Output (`ListConnectionsOutput`)

```json
{
  "connections": [
    {"name": "default", "host": "prod.trino.example.com", "port": 443, "catalog": "hive", "ssl": true, "is_default": true},
    {"name": "staging", "host": "staging.trino.example.com", "port": 443, "catalog": "hive", "ssl": true, "is_default": false}
  ],
  "count": 2
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

An error result has `isError: true` and a text content block with a human-readable message.

When Trino fails to run a statement, `trino_query` and `trino_execute` also put a classification of the failure in `structuredContent.error`. The text content keeps the same message (`Query failed: ...` or `Execution failed: ...`). The other `QueryOutput` fields are present with zero values (`columns` and `rows` are `null`, `row_count` is `0`), so check `isError` rather than reading an empty result as zero rows.

```json
{
  "columns": null,
  "rows": null,
  "row_count": 0,
  "stats": {"row_count": 0, "truncated": false, "duration_ms": 0},
  "error": {
    "code": "trino_query_failed",
    "category": "upstream_unavailable",
    "retryable": true,
    "message": "EXTERNAL: The connection attempt failed.",
    "trino": {"error_type": "EXTERNAL", "error_name": "JDBC_ERROR", "error_code": 67108864, "http_status": 200},
    "transport": null
  }
}
```

| Field | Meaning |
|-------|---------|
| `category` | `upstream_unavailable` (Trino, a source behind it, or the network failed), `client_input` (the statement is wrong), or `internal` (anything else) |
| `retryable` | Whether the same statement is expected to succeed if run again later |
| `message` | Trino's message when Trino reported one, otherwise the error text |
| `trino` | The error Trino reported (`error_type`, `error_name`, `error_code`, `sql_state` when sent, `http_status`), or `null` |
| `transport` | For a failure before Trino reported an error: `kind` (`timeout`, `connection_refused`, `connection_reset`, `dns`, `network`, `http_status`), `http_status` for the last kind, and `detail`. Otherwise `null` |

`structuredContent.error` is absent when the failure was not the query's: a rejected or missing argument, a cancelled query, or a call whose own context ended. `client.Classify` in `pkg/client` applies the same classification to any error the client returns.

How failures are classified:

| Failure | Category | Retryable |
|---------|----------|-----------|
| Deadline exceeded, connection refused or reset, DNS failure, other network error, HTTP 429/502/503/504 | `upstream_unavailable` | yes |
| Other HTTP status with no Trino error (401, a redirect) | `internal` | no |
| `USER_ERROR` | `client_input` | no |
| `INSUFFICIENT_RESOURCES` | `upstream_unavailable` | yes |
| `INTERNAL_ERROR` naming a node or transport failure (`SERVER_SHUTTING_DOWN`, `SERVER_STARTING_UP`, `NO_NODES_AVAILABLE`, `REMOTE_HOST_GONE`, `REMOTE_TASK_ERROR`, `REMOTE_TASK_FAILED`, `ABANDONED_TASK`, `PAGE_TRANSPORT_ERROR`, `PAGE_TRANSPORT_TIMEOUT`, `TOO_MANY_REQUESTS_FAILED`) | `upstream_unavailable` | yes |
| Other `INTERNAL_ERROR` | `internal` | no |
| `EXTERNAL` from a source that could not be reached | `upstream_unavailable` | yes |
| `EXTERNAL` from a source that rejected the statement | `client_input` | no |
| Other `EXTERNAL` | `upstream_unavailable` | no |

An `EXTERNAL` error is decided by its SQLSTATE class when one is present (`08` is unreachable; `22`, `23` and `42` are rejected). Trino servers send `sqlState` as null, so in practice the decision comes from the Java exception types in the failure's cause chain: a `java.net` connection exception or a `java.sql` connection exception means unreachable, and `SQLDataException`, `SQLIntegrityConstraintViolationException` or `SQLSyntaxErrorException` means rejected. PostgreSQL's "The connection attempt failed." message also counts as unreachable. When both kinds appear in the chain, the rejection wins.

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
