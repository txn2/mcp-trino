package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrorResult creates an error CallToolResult with the given message.
// Use this in custom tool handlers to return errors in MCP format.
func ErrorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}

// TextResult creates a successful CallToolResult with text content.
// Use this in custom tool handlers to return text results.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}
