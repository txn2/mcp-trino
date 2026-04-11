package tools

// defaultDescriptions holds the default description for each built-in tool.
// These are used when no override is provided via WithDescription or WithDescriptions.
var defaultDescriptions = map[ToolName]string{
	ToolQuery: "Execute a read-only SQL query against Trino and return results. " +
		"Only SELECT, SHOW, DESCRIBE, EXPLAIN, and WITH statements are allowed. " +
		"Use the table path format catalog.schema.table. Results are limited to prevent " +
		"excessive data transfer — use the limit parameter to control result size. " +
		"For large tables, always include WHERE clauses to filter results. " +
		"Consider using trino_explain first for expensive queries. " +
		"Results are returned as JSON by default. Pass format=csv for CSV output " +
		"(more token-efficient for large result sets) or format=markdown for a pipe-table. " +
		"Set unwrap_json=true to automatically parse single-row, single-VARCHAR-column " +
		"results containing JSON (common with table functions like raw_query). " +
		"For write operations (INSERT, CREATE, etc.), use trino_execute instead.",

	ToolExecute: "Execute a SQL statement against Trino, including write operations " +
		"(INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, etc.). Returns affected row " +
		"counts or query results. Use trino_query for read-only SELECT queries. " +
		"This tool should be used when you need to modify data or schema. " +
		"Results are returned as JSON by default. Pass format=csv for CSV output " +
		"(more token-efficient for large result sets) or format=markdown for a pipe-table. " +
		"Set unwrap_json=true to automatically parse single-row, single-VARCHAR-column " +
		"results containing JSON (common with table functions like raw_query).",

	ToolExplain: "Get the execution plan for a SQL query to understand performance characteristics " +
		"before running expensive queries. Use this when querying large tables (millions of " +
		"rows) to verify the query plan uses appropriate filters. Also useful for debugging " +
		"slow queries or understanding join strategies.",

	ToolBrowse: "Browse the Trino catalog hierarchy. " +
		"Omit all parameters to list catalogs. " +
		"Provide catalog to list schemas. " +
		"Provide catalog and schema to list tables (with optional pattern filter).",

	ToolDescribeTable: "Get detailed table information with columns, types, and optional sample data. " +
		"Set include_sample=true to see actual data values, which helps understand column " +
		"meaning and data formats. This is the richest single-call way to understand a " +
		"table's structure. Requires catalog, schema, and table name — use trino_browse " +
		"to discover available tables if needed.",

	ToolListConnections: "List all configured Trino server connections. " +
		"Use this to discover available connections before querying specific servers. " +
		"Pass the connection name to other tools via the 'connection' parameter.",
}

// DefaultDescription returns the default description for a tool.
// Returns an empty string for unknown tool names.
func DefaultDescription(name ToolName) string {
	return defaultDescriptions[name]
}

// getDescription resolves the description for a tool using the priority chain:
// 1. Per-registration override (cfg.description) — highest priority
// 2. Toolkit-level override (t.descriptions) — medium priority
// 3. Default description — lowest priority.
func (t *Toolkit) getDescription(name ToolName, cfg *toolConfig) string {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.description != nil {
		return *cfg.description
	}

	// Toolkit-level override (medium priority)
	if desc, ok := t.descriptions[name]; ok {
		return desc
	}

	// Default description (lowest priority)
	return defaultDescriptions[name]
}
