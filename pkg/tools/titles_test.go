package tools

import "testing"

func TestDefaultTitle(t *testing.T) {
	tests := []struct {
		name    ToolName
		wantNon bool // expect non-empty
	}{
		{ToolQuery, true},
		{ToolExecute, true},
		{ToolExplain, true},
		{ToolListCatalogs, true},
		{ToolListSchemas, true},
		{ToolListTables, true},
		{ToolDescribeTable, true},
		{ToolListConnections, true},
		{ToolName("unknown_tool"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			title := DefaultTitle(tt.name)
			if tt.wantNon && title == "" {
				t.Errorf("expected non-empty title for %s", tt.name)
			}
			if !tt.wantNon && title != "" {
				t.Errorf("expected empty title for %s, got %q", tt.name, title)
			}
		})
	}
}

func TestDefaultTitles_AllToolsCovered(t *testing.T) {
	for _, name := range AllTools() {
		if DefaultTitle(name) == "" {
			t.Errorf("tool %s has no default title", name)
		}
	}
}

func TestDefaultTitles_ExpectedValues(t *testing.T) {
	tests := []struct {
		name  ToolName
		title string
	}{
		{ToolQuery, "Execute SQL Query"},
		{ToolExecute, "Execute SQL (Write)"},
		{ToolExplain, "Explain Query Plan"},
		{ToolListCatalogs, "List Catalogs"},
		{ToolListSchemas, "List Schemas"},
		{ToolListTables, "List Tables"},
		{ToolDescribeTable, "Describe Table"},
		{ToolListConnections, "List Connections"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			got := DefaultTitle(tt.name)
			if got != tt.title {
				t.Errorf("DefaultTitle(%s) = %q, want %q", tt.name, got, tt.title)
			}
		})
	}
}

func TestGetTitle_Priority(t *testing.T) {
	tests := []struct {
		name          string
		toolkitTitles map[ToolName]string
		cfgTitle      *string
		toolName      ToolName
		wantTitle     string
	}{
		{
			name:          "default only",
			toolkitTitles: nil,
			cfgTitle:      nil,
			toolName:      ToolQuery,
			wantTitle:     defaultTitles[ToolQuery],
		},
		{
			name:          "toolkit override",
			toolkitTitles: map[ToolName]string{ToolQuery: "Toolkit Title"},
			cfgTitle:      nil,
			toolName:      ToolQuery,
			wantTitle:     "Toolkit Title",
		},
		{
			name:          "per-registration override",
			toolkitTitles: nil,
			cfgTitle:      strPtr("Per-reg Title"),
			toolName:      ToolQuery,
			wantTitle:     "Per-reg Title",
		},
		{
			name:          "per-registration beats toolkit",
			toolkitTitles: map[ToolName]string{ToolQuery: "Toolkit Title"},
			cfgTitle:      strPtr("Per-reg Title"),
			toolName:      ToolQuery,
			wantTitle:     "Per-reg Title",
		},
		{
			name:          "toolkit override for one tool, default for another",
			toolkitTitles: map[ToolName]string{ToolQuery: "Custom query title"},
			cfgTitle:      nil,
			toolName:      ToolExplain,
			wantTitle:     defaultTitles[ToolExplain],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &Toolkit{
				titles: make(map[ToolName]string),
			}
			if tt.toolkitTitles != nil {
				tk.titles = tt.toolkitTitles
			}

			var cfg *toolConfig
			if tt.cfgTitle != nil {
				cfg = &toolConfig{title: tt.cfgTitle}
			}

			got := tk.getTitle(tt.toolName, cfg)
			if got != tt.wantTitle {
				t.Errorf("getTitle() = %q, want %q", got, tt.wantTitle)
			}
		})
	}
}

func TestGetTitle_NilConfig(t *testing.T) {
	tk := &Toolkit{
		titles: make(map[ToolName]string),
	}

	got := tk.getTitle(ToolQuery, nil)
	if got != defaultTitles[ToolQuery] {
		t.Errorf("expected default title, got %q", got)
	}
}
