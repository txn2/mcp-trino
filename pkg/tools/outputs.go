package tools

// QueryOutput defines the structured output of the trino_query tool.
type QueryOutput struct {
	Columns  []QueryColumn    `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
	Stats    QueryStats       `json:"stats"`
}

// QueryColumn describes a column in the query result.
type QueryColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryStats provides execution statistics for a query.
type QueryStats struct {
	RowCount     int   `json:"row_count"`
	Truncated    bool  `json:"truncated"`
	LimitApplied int   `json:"limit_applied,omitempty"`
	DurationMs   int64 `json:"duration_ms"`
}

// ExplainOutput defines the structured output of the trino_explain tool.
type ExplainOutput struct {
	Plan string `json:"plan"`
	Type string `json:"type"`
}

// ListCatalogsOutput defines the structured output of the trino_list_catalogs tool.
type ListCatalogsOutput struct {
	Catalogs []string `json:"catalogs"`
	Count    int      `json:"count"`
}

// ListSchemasOutput defines the structured output of the trino_list_schemas tool.
type ListSchemasOutput struct {
	Catalog string   `json:"catalog"`
	Schemas []string `json:"schemas"`
	Count   int      `json:"count"`
}

// ListTablesOutput defines the structured output of the trino_list_tables tool.
type ListTablesOutput struct {
	Catalog string   `json:"catalog"`
	Schema  string   `json:"schema"`
	Tables  []string `json:"tables"`
	Count   int      `json:"count"`
	Pattern string   `json:"pattern,omitempty"`
}

// DescribeTableOutput defines the structured output of the trino_describe_table tool.
type DescribeTableOutput struct {
	Catalog string           `json:"catalog"`
	Schema  string           `json:"schema"`
	Table   string           `json:"table"`
	Columns []DescribeColumn `json:"columns"`
	Count   int              `json:"column_count"`
	Sample  []map[string]any `json:"sample,omitempty"`
}

// DescribeColumn describes a column in the table.
type DescribeColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable,omitempty"`
	Comment  string `json:"comment,omitempty"`
}
