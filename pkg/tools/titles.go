package tools

// defaultTitles holds the default human-readable display title for each built-in tool.
// These are shown in MCP clients (e.g. Claude Desktop) as the visible tool name.
// When no override is provided via WithTitle or WithTitles, these are used.
var defaultTitles = map[ToolName]string{
	ToolQuery:           "Execute SQL Query",
	ToolExecute:         "Execute SQL (Write)",
	ToolExplain:         "Explain Query Plan",
	ToolBrowse:          "Browse Catalog",
	ToolDescribeTable:   "Describe Table",
	ToolListConnections: "List Connections",
}

// DefaultTitle returns the default human-readable title for a tool.
// Returns an empty string for unknown tool names.
func DefaultTitle(name ToolName) string {
	return defaultTitles[name]
}

// getTitle resolves the title for a tool using the priority chain:
// 1. Per-registration override (cfg.title) — highest priority
// 2. Toolkit-level override (t.titles) — medium priority
// 3. Default title — lowest priority.
func (t *Toolkit) getTitle(name ToolName, cfg *toolConfig) string {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.title != nil {
		return *cfg.title
	}

	// Toolkit-level override (medium priority)
	if title, ok := t.titles[name]; ok {
		return title
	}

	// Default title (lowest priority)
	return defaultTitles[name]
}
