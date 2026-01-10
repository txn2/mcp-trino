package extensions

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// ErrorEnricher adds helpful hints to error messages.
// This improves the AI experience by suggesting next steps when errors occur.
type ErrorEnricher struct {
	hints map[string]string
}

// NewErrorEnricher creates a new error enricher with default hints.
func NewErrorEnricher() *ErrorEnricher {
	return &ErrorEnricher{
		hints: map[string]string{
			// Table/schema/catalog not found
			"table not found":   "Hint: Use trino_list_tables to see available tables in the schema.",
			"schema not found":  "Hint: Use trino_list_schemas to see available schemas in the catalog.",
			"catalog not found": "Hint: Use trino_list_catalogs to see available catalogs.",
			"does not exist":    "Hint: Verify the object name and use the list tools to explore available objects.",

			// Permission errors
			"access denied":     "Hint: Check your Trino user permissions for this operation.",
			"permission denied": "Hint: Check your Trino user permissions for this operation.",
			"not authorized":    "Hint: Check your Trino user permissions for this operation.",

			// Query errors
			"syntax error":     "Hint: Check your SQL syntax. Trino uses ANSI SQL with some extensions.",
			"column not found": "Hint: Use trino_describe_table to see available columns.",
			"ambiguous column": "Hint: Qualify the column name with its table alias.",
			"type mismatch":    "Hint: Check data types with trino_describe_table and use CAST() if needed.",

			// Connection errors
			"connection refused": "Hint: Verify the Trino server is running and accessible.",
			"timeout":            "Hint: The query took too long. Try adding LIMIT or optimizing the query.",

			// Read-only mode
			"modification statements are not allowed": "Hint: This server is in read-only mode. Only SELECT queries are allowed.",
		},
	}
}

// Transform adds helpful hints to error results.
func (ee *ErrorEnricher) Transform(
	_ context.Context,
	_ tools.ToolName,
	result *mcp.CallToolResult,
) (*mcp.CallToolResult, error) {
	if result == nil || !result.IsError {
		return result, nil
	}

	// Extract error text
	errorText := ""
	for _, content := range result.Content {
		if tc, ok := content.(*mcp.TextContent); ok {
			errorText = tc.Text
			break
		}
	}

	if errorText == "" {
		return result, nil
	}

	// Find matching hint
	lowerError := strings.ToLower(errorText)
	for pattern, hint := range ee.hints {
		if strings.Contains(lowerError, pattern) {
			// Append hint to error message
			for i, content := range result.Content {
				if tc, ok := content.(*mcp.TextContent); ok {
					tc.Text = tc.Text + "\n\n" + hint
					result.Content[i] = tc
					return result, nil
				}
			}
		}
	}

	return result, nil
}

// AddHint adds a custom hint for a specific error pattern.
// Pattern matching is case-insensitive.
func (ee *ErrorEnricher) AddHint(pattern, hint string) {
	ee.hints[strings.ToLower(pattern)] = hint
}

// Verify ErrorEnricher implements ResultTransformer.
var _ tools.ResultTransformer = (*ErrorEnricher)(nil)
