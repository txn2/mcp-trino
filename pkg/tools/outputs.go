package tools

import "github.com/txn2/mcp-trino/pkg/client"

// QueryOutput defines the structured output of the trino_query tool.
type QueryOutput struct {
	Columns  []QueryColumn    `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
	Stats    QueryStats       `json:"stats"`

	// Error classifies the failure when the query failed. It is absent on
	// success, and on a failure that was not the query's: a rejected input,
	// a cancellation.
	Error *QueryError `json:"error,omitempty"`
}

// QueryErrorCode is the Code of every QueryError.
const QueryErrorCode = "trino_query_failed"

// QueryError is the classification of a failed query, as client.Classify
// reports it, in a trino_query or trino_execute error result.
type QueryError struct {
	Code      string                       `json:"code"`
	Category  client.ErrorCategory         `json:"category"`
	Retryable bool                         `json:"retryable"`
	Message   string                       `json:"message"`
	Trino     *client.TrinoErrorDetail     `json:"trino"`
	Transport *client.TransportErrorDetail `json:"transport"`
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

// BrowseOutput defines the structured output of the trino_browse tool.
type BrowseOutput struct {
	Level   string   `json:"level"`
	Catalog string   `json:"catalog,omitempty"`
	Schema  string   `json:"schema,omitempty"`
	Items   []string `json:"items"`
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
