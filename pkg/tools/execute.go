package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// ExecuteInput defines the input for the trino_execute tool.
type ExecuteInput struct {
	// SQL is the SQL statement to execute (including write operations).
	SQL string `json:"sql" jsonschema_description:"The SQL statement to execute (SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, etc.)"`

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

// registerExecuteTool adds the trino_execute tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerExecuteTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		executeInput, ok := input.(ExecuteInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleExecute(ctx, req, executeInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolExecute, baseHandler, cfg)

	// Register with MCP using typed handler that calls wrapped handler
	mcp.AddTool(server, &mcp.Tool{
		Name:        string(ToolExecute),
		Description: t.getDescription(ToolExecute, cfg),
		Annotations: t.getAnnotations(ToolExecute, cfg),
		Icons:       t.getIcons(ToolExecute, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ExecuteInput) (*mcp.CallToolResult, *QueryOutput, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*QueryOutput); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

func (t *Toolkit) handleExecute(ctx context.Context, _ *mcp.CallToolRequest, input ExecuteInput) (*mcp.CallToolResult, any, error) {
	// Validate SQL is provided
	if input.SQL == "" {
		return ErrorResult("sql parameter is required"), nil, nil
	}

	// Apply query interceptors (no read-only enforcement — that's the point of trino_execute)
	sql, err := t.InterceptSQL(ctx, input.SQL, ToolExecute)
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

	// Send progress notification: executing statement
	notifier := GetProgressNotifier(ctx)
	notifyProgress(ctx, notifier, 0, 3, "Executing statement...")

	// Execute query
	opts := client.QueryOptions{
		Limit:   limit,
		Timeout: timeout,
	}

	result, err := trinoClient.Query(ctx, sql, opts)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Execution failed: %v", err)), nil, nil
	}

	// Send progress notification: formatting results
	notifyProgress(ctx, notifier, 1, 3,
		fmt.Sprintf("Statement returned %d rows, formatting...", result.Stats.RowCount))

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

	// Send progress notification: complete
	notifyProgress(ctx, notifier, 2, 3, "Execution complete")

	// Build structured output (reuse QueryOutput — same result shape)
	queryOutput := buildQueryOutput(result)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, &queryOutput, nil
}
