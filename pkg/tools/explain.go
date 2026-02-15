package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// ExplainInput defines the input for the trino_explain tool.
type ExplainInput struct {
	// SQL is the SQL query to explain.
	SQL string `json:"sql" jsonschema_description:"The SQL query to explain"`

	// Type is the explain type: logical, distributed, io, or validate.
	Type string `json:"type,omitempty" jsonschema_description:"Explain type: logical (default), distributed, io, or validate"`

	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerExplainTool adds the trino_explain tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerExplainTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		explainInput, ok := input.(ExplainInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleExplain(ctx, req, explainInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolExplain, baseHandler, cfg)

	// Register with MCP using typed handler that calls wrapped handler
	mcp.AddTool(server, &mcp.Tool{
		Name: "trino_explain",
		Description: "Get the execution plan for a SQL query to understand performance characteristics " +
			"before running expensive queries. Use this when querying large tables (millions of " +
			"rows) to verify the query plan uses appropriate filters. Also useful for debugging " +
			"slow queries or understanding join strategies.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

func (t *Toolkit) handleExplain(ctx context.Context, _ *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, any, error) {
	if input.SQL == "" {
		return ErrorResult("sql parameter is required"), nil, nil
	}

	// Apply query interceptors
	sql, err := t.InterceptSQL(ctx, input.SQL, ToolExplain)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Query rejected: %v", err)), nil, nil
	}

	// Map type string to ExplainType
	var explainType client.ExplainType
	switch input.Type {
	case "distributed":
		explainType = client.ExplainDistributed
	case "io":
		explainType = client.ExplainIO
	case "validate":
		explainType = client.ExplainValidate
	default:
		explainType = client.ExplainLogical
	}

	// Get client for the specified connection
	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	result, err := trinoClient.Explain(ctx, sql, explainType)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Explain failed: %v", err)), nil, nil
	}

	output := fmt.Sprintf("## Execution Plan (%s)\n\n```\n%s\n```", result.Type, result.Plan)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}
