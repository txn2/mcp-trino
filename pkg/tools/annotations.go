package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// defaultAnnotations holds the default annotations for each built-in tool.
// These follow the MCP specification:
//   - ReadOnlyHint (bool, default false): tool does not modify state
//   - DestructiveHint (*bool, default true): tool may destructively update
//   - IdempotentHint (bool, default false): repeated calls produce same result
//   - OpenWorldHint (*bool, default true): tool interacts with external entities
//
// All Trino tools communicate with an external Trino cluster, so OpenWorldHint
// is true for all tools — this correctly signals to MCP clients that these
// tools interact with entities outside the server's direct control.
var defaultAnnotations = map[ToolName]*mcp.ToolAnnotations{
	ToolQuery: {
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		DestructiveHint: boolPtr(false),
		OpenWorldHint:   boolPtr(true),
	},
	ToolExecute: {
		DestructiveHint: boolPtr(true),
		OpenWorldHint:   boolPtr(true),
	},
	ToolExplain: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolListCatalogs: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolListSchemas: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolListTables: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolDescribeTable: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
	ToolListConnections: {
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	},
}

// DefaultAnnotations returns the default annotations for a tool.
// Returns nil for unknown tool names.
func DefaultAnnotations(name ToolName) *mcp.ToolAnnotations {
	return defaultAnnotations[name]
}

// getAnnotations resolves the annotations for a tool using the priority chain:
// 1. Per-registration override (cfg.annotations) — highest priority
// 2. Toolkit-level override (t.annotations) — medium priority
// 3. Default annotations — lowest priority.
func (t *Toolkit) getAnnotations(name ToolName, cfg *toolConfig) *mcp.ToolAnnotations {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.annotations != nil {
		return cfg.annotations
	}

	// Toolkit-level override (medium priority)
	if ann, ok := t.annotations[name]; ok {
		return ann
	}

	// Default annotations (lowest priority)
	return defaultAnnotations[name]
}
