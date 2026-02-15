package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// QueryInput defines the input for the trino_query tool.
type QueryInput struct {
	// SQL is the SQL query to execute.
	SQL string `json:"sql" jsonschema_description:"The SQL query to execute"`

	// Limit is the maximum number of rows to return. Default: 1000, Max: 10000.
	Limit int `json:"limit,omitempty" jsonschema_description:"Maximum rows to return (default: 1000, max: 10000)"`

	// TimeoutSeconds is the query timeout in seconds. Default: 120, Max: 300.
	TimeoutSeconds int `json:"timeout_seconds,omitempty" jsonschema_description:"Query timeout in seconds (default: 120, max: 300)"`

	// Format is the output format: json (default), csv, or markdown.
	Format string `json:"format,omitempty" jsonschema_description:"Output format: json, csv, or markdown (default: json)"`

	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerQueryTool adds the trino_query tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerQueryTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		queryInput, ok := input.(QueryInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleQuery(ctx, req, queryInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolQuery, baseHandler, cfg)

	// Register with MCP using typed handler that calls wrapped handler
	mcp.AddTool(server, &mcp.Tool{
		Name:        string(ToolQuery),
		Description: t.getDescription(ToolQuery, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

func (t *Toolkit) handleQuery(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
	// Validate SQL is provided
	if input.SQL == "" {
		return ErrorResult("sql parameter is required"), nil, nil
	}

	// Apply query interceptors
	sql, err := t.InterceptSQL(ctx, input.SQL, ToolQuery)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Query rejected: %v", err)), nil, nil
	}

	// Apply limits
	limit := input.Limit
	if limit <= 0 {
		limit = t.config.DefaultLimit
	}
	if limit > t.config.MaxLimit {
		limit = t.config.MaxLimit
	}

	// Apply timeout
	timeout := time.Duration(input.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = t.config.DefaultTimeout
	}
	if timeout > t.config.MaxTimeout {
		timeout = t.config.MaxTimeout
	}

	// Get client for the specified connection
	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	// Execute query
	opts := client.QueryOptions{
		Limit:   limit,
		Timeout: timeout,
	}

	result, err := trinoClient.Query(ctx, sql, opts)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Query failed: %v", err)), nil, nil
	}

	// Format output
	format := input.Format
	if format == "" {
		format = "json"
	}

	var output string
	switch format {
	case "csv":
		output = formatCSV(result)
	case "markdown":
		output = formatMarkdown(result)
	default:
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return ErrorResult(fmt.Sprintf("Failed to marshal result: %v", err)), nil, nil
		}
		output = string(data)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

// formatCSV formats query results as CSV.
func formatCSV(result *client.QueryResult) string {
	if len(result.Columns) == 0 {
		return ""
	}

	var output string

	// Header row
	for i, col := range result.Columns {
		if i > 0 {
			output += ","
		}
		output += escapeCSV(col.Name)
	}
	output += "\n"

	// Data rows
	for _, row := range result.Rows {
		for i, col := range result.Columns {
			if i > 0 {
				output += ","
			}
			if val, ok := row[col.Name]; ok {
				output += escapeCSV(fmt.Sprintf("%v", val))
			}
		}
		output += "\n"
	}

	// Stats footer
	output += fmt.Sprintf("\n# %d rows returned", result.Stats.RowCount)
	if result.Stats.Truncated {
		output += fmt.Sprintf(" (truncated at limit %d)", result.Stats.LimitApplied)
	}
	output += fmt.Sprintf(", executed in %dms", result.Stats.DurationMs)

	return output
}

// formatMarkdown formats query results as a Markdown table.
func formatMarkdown(result *client.QueryResult) string {
	if len(result.Columns) == 0 {
		return "No results"
	}

	var output string

	// Header row
	output += "|"
	for _, col := range result.Columns {
		output += " " + col.Name + " |"
	}
	output += "\n|"

	// Separator row
	for range result.Columns {
		output += " --- |"
	}
	output += "\n"

	// Data rows
	for _, row := range result.Rows {
		output += "|"
		for _, col := range result.Columns {
			val := ""
			if v, ok := row[col.Name]; ok && v != nil {
				val = fmt.Sprintf("%v", v)
			}
			output += " " + val + " |"
		}
		output += "\n"
	}

	// Stats footer
	output += fmt.Sprintf("\n*%d rows returned", result.Stats.RowCount)
	if result.Stats.Truncated {
		output += fmt.Sprintf(" (truncated at limit %d)", result.Stats.LimitApplied)
	}
	output += fmt.Sprintf(", executed in %dms*", result.Stats.DurationMs)

	return output
}

// escapeCSV escapes a value for CSV output.
func escapeCSV(s string) string {
	needsQuoting := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}

	// Escape quotes and wrap in quotes
	result := "\""
	for _, c := range s {
		if c == '"' {
			result += "\"\""
		} else {
			result += string(c)
		}
	}
	result += "\""
	return result
}
