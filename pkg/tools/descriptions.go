package tools

// defaultDescriptions holds the default description for each built-in tool.
// These are used when no override is provided via WithDescription or WithDescriptions.
var defaultDescriptions = map[ToolName]string{
	ToolQuery: "Execute a SELECT query against Trino and return results. Use the table path format " +
		"catalog.schema.table. Results are limited to prevent excessive data transfer — use " +
		"the limit parameter to control result size. For large tables, always include WHERE " +
		"clauses to filter results. Consider using trino_explain first for expensive queries.",

	ToolExplain: "Get the execution plan for a SQL query to understand performance characteristics " +
		"before running expensive queries. Use this when querying large tables (millions of " +
		"rows) to verify the query plan uses appropriate filters. Also useful for debugging " +
		"slow queries or understanding join strategies.",

	ToolListCatalogs: "List all available catalogs in the Trino cluster. Catalogs are the top-level " +
		"containers for schemas and tables.",

	ToolListSchemas: "List all schemas in a catalog. Schemas are containers for tables within a catalog.",

	ToolListTables: "List all tables in a schema. Optionally filter by a LIKE pattern.",

	ToolDescribeTable: "Get detailed table information with columns, types, and optional sample data. " +
		"Set include_sample=true to see actual data values, which helps understand column " +
		"meaning and data formats. This is the richest single-call way to understand a " +
		"table's structure. Requires catalog, schema, and table name — use trino_list_tables " +
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
