package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResultTransformerFunc(t *testing.T) {
	transformer := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		result *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		// Add a marker to the result.
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp.TextContent); ok {
				tc.Text += " [transformed]"
			}
		}
		return result, nil
	})

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "original"},
		},
	}

	transformed, err := transformer.Transform(context.Background(), ToolQuery, result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	text := transformed.Content[0].(*mcp.TextContent).Text
	if text != "original [transformed]" {
		t.Errorf("expected transformed text, got: %s", text)
	}
}

func TestResultTransformerFunc_Error(t *testing.T) {
	expectedErr := errors.New("transform error")
	transformer := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		_ *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	})

	_, err := transformer.Transform(context.Background(), ToolQuery, &mcp.CallToolResult{})
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestTransformerChain_Transform(t *testing.T) {
	// First transformer adds [1].
	t1 := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		result *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp.TextContent); ok {
				tc.Text += "[1]"
			}
		}
		return result, nil
	})

	// Second transformer adds [2].
	t2 := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		result *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp.TextContent); ok {
				tc.Text += "[2]"
			}
		}
		return result, nil
	})

	chain := NewTransformerChain(t1, t2)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "base"},
		},
	}

	transformed, err := chain.Transform(context.Background(), ToolQuery, result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	text := transformed.Content[0].(*mcp.TextContent).Text
	if text != "base[1][2]" {
		t.Errorf("expected chained transformations, got: %s", text)
	}
}

func TestTransformerChain_Transform_Error(t *testing.T) {
	expectedErr := errors.New("transform error")

	t1 := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		_ *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		return nil, expectedErr
	})

	t2 := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		result *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		t.Error("t2 should not be called")
		return result, nil
	})

	chain := NewTransformerChain(t1, t2)

	_, err := chain.Transform(context.Background(), ToolQuery, &mcp.CallToolResult{})
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestTransformerChain_Append(t *testing.T) {
	chain := NewTransformerChain()
	if chain.Len() != 0 {
		t.Errorf("expected empty chain, got %d", chain.Len())
	}

	transformer := ResultTransformerFunc(func(
		_ context.Context,
		_ ToolName,
		result *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		return result, nil
	})
	chain.Append(transformer)

	if chain.Len() != 1 {
		t.Errorf("expected chain length 1, got %d", chain.Len())
	}
}

func TestTransformerChain_ToolNamePassed(t *testing.T) {
	var receivedToolName ToolName

	transformer := ResultTransformerFunc(func(
		_ context.Context,
		toolName ToolName,
		result *mcp.CallToolResult,
	) (*mcp.CallToolResult, error) {
		receivedToolName = toolName
		return result, nil
	})

	chain := NewTransformerChain(transformer)
	_, _ = chain.Transform(context.Background(), ToolDescribeTable, &mcp.CallToolResult{})

	if receivedToolName != ToolDescribeTable {
		t.Errorf("expected ToolDescribeTable, got %v", receivedToolName)
	}
}
