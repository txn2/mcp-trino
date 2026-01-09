package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResultTransformer modifies tool results after execution.
// Use transformers for redaction, formatting changes, enrichment,
// or any result-specific transformation.
type ResultTransformer interface {
	// Transform modifies the result after tool execution.
	// Called after all middleware After hooks have completed.
	// Return an error to replace the result with an error result.
	Transform(ctx context.Context, toolName ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error)
}

// ResultTransformerFunc allows using a function as a transformer.
type ResultTransformerFunc func(ctx context.Context, toolName ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error)

// Transform implements ResultTransformer.
func (f ResultTransformerFunc) Transform(ctx context.Context, toolName ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	return f(ctx, toolName, result)
}

// TransformerChain chains multiple transformers together.
// Transformers are executed in order.
type TransformerChain struct {
	transformers []ResultTransformer
}

// NewTransformerChain creates a transformer chain from the given transformers.
func NewTransformerChain(transformers ...ResultTransformer) *TransformerChain {
	return &TransformerChain{transformers: transformers}
}

// Transform runs all transformers in order.
// Stops and returns error if any transformer returns an error.
func (tc *TransformerChain) Transform(ctx context.Context, toolName ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	var err error
	for _, t := range tc.transformers {
		result, err = t.Transform(ctx, toolName, result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Append adds transformers to the chain.
func (tc *TransformerChain) Append(transformers ...ResultTransformer) {
	tc.transformers = append(tc.transformers, transformers...)
}

// Len returns the number of transformers in the chain.
func (tc *TransformerChain) Len() int {
	return len(tc.transformers)
}
