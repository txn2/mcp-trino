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
}

// registerExplainTool adds the trino_explain tool to the server.
func (t *Toolkit) registerExplainTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "trino_explain",
		Description: "Get the execution plan for a SQL query. " +
			"Use this to understand how Trino will execute a query and identify potential performance issues.",
	}, t.handleExplain)
}

func (t *Toolkit) handleExplain(ctx context.Context, _ *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, any, error) {
	if input.SQL == "" {
		return errorResult("sql parameter is required"), nil, nil
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

	result, err := t.client.Explain(ctx, input.SQL, explainType)
	if err != nil {
		return errorResult(fmt.Sprintf("Explain failed: %v", err)), nil, nil
	}

	output := fmt.Sprintf("## Execution Plan (%s)\n\n```\n%s\n```", result.Type, result.Plan)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}
