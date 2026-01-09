package tools

// ToolName identifies a tool for registration and middleware targeting.
type ToolName string

// Built-in tool names.
const (
	ToolQuery         ToolName = "trino_query"
	ToolExplain       ToolName = "trino_explain"
	ToolListCatalogs  ToolName = "trino_list_catalogs"
	ToolListSchemas   ToolName = "trino_list_schemas"
	ToolListTables    ToolName = "trino_list_tables"
	ToolDescribeTable ToolName = "trino_describe_table"
)

// AllTools returns all built-in tool names.
func AllTools() []ToolName {
	return []ToolName{
		ToolQuery,
		ToolExplain,
		ToolListCatalogs,
		ToolListSchemas,
		ToolListTables,
		ToolDescribeTable,
	}
}

// QueryTools returns tools that execute SQL queries.
// These tools are subject to query interception.
func QueryTools() []ToolName {
	return []ToolName{
		ToolQuery,
		ToolExplain,
	}
}

// SchemaTools returns tools that query schema metadata.
func SchemaTools() []ToolName {
	return []ToolName{
		ToolListCatalogs,
		ToolListSchemas,
		ToolListTables,
		ToolDescribeTable,
	}
}

// String returns the string representation of the tool name.
func (n ToolName) String() string {
	return string(n)
}
