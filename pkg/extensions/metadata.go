package extensions

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// MetadataEnricher appends execution metadata to tool results.
// This provides visibility into tool execution for debugging and monitoring.
type MetadataEnricher struct {
	// startTimes tracks when each tool started (keyed by context pointer for uniqueness)
	// In production, you'd use context values instead
}

// NewMetadataEnricher creates a new metadata enricher.
func NewMetadataEnricher() *MetadataEnricher {
	return &MetadataEnricher{}
}

// Transform appends execution metadata footer to results.
func (me *MetadataEnricher) Transform(
	_ context.Context,
	toolName tools.ToolName,
	result *mcp.CallToolResult,
) (*mcp.CallToolResult, error) {
	if result == nil {
		return nil, nil
	}

	// Build metadata footer
	metadata := fmt.Sprintf("\n---\nTool: %s | Time: %s",
		toolName,
		time.Now().Format(time.RFC3339),
	)

	// Append metadata to the last text content
	for i := len(result.Content) - 1; i >= 0; i-- {
		if tc, ok := result.Content[i].(*mcp.TextContent); ok {
			tc.Text += metadata
			return result, nil
		}
	}

	// If no text content found, add it as new content
	result.Content = append(result.Content, &mcp.TextContent{Text: metadata})
	return result, nil
}

// Verify MetadataEnricher implements ResultTransformer.
var _ tools.ResultTransformer = (*MetadataEnricher)(nil)
