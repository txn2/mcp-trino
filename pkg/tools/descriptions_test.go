package tools

import "testing"

func TestDefaultDescription(t *testing.T) {
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
			desc := DefaultDescription(tt.name)
			if tt.wantNon && desc == "" {
				t.Errorf("expected non-empty description for %s", tt.name)
			}
			if !tt.wantNon && desc != "" {
				t.Errorf("expected empty description for %s, got %q", tt.name, desc)
			}
		})
	}
}

func TestDefaultDescriptions_AllToolsCovered(t *testing.T) {
	for _, name := range AllTools() {
		if DefaultDescription(name) == "" {
			t.Errorf("tool %s has no default description", name)
		}
	}
}

func TestGetDescription_Priority(t *testing.T) {
	tests := []struct {
		name         string
		toolkitDescs map[ToolName]string
		cfgDesc      *string
		toolName     ToolName
		wantDesc     string
	}{
		{
			name:         "default only",
			toolkitDescs: nil,
			cfgDesc:      nil,
			toolName:     ToolQuery,
			wantDesc:     defaultDescriptions[ToolQuery],
		},
		{
			name:         "toolkit override",
			toolkitDescs: map[ToolName]string{ToolQuery: "Toolkit override"},
			cfgDesc:      nil,
			toolName:     ToolQuery,
			wantDesc:     "Toolkit override",
		},
		{
			name:         "per-registration override",
			toolkitDescs: nil,
			cfgDesc:      strPtr("Per-reg override"),
			toolName:     ToolQuery,
			wantDesc:     "Per-reg override",
		},
		{
			name:         "per-registration beats toolkit",
			toolkitDescs: map[ToolName]string{ToolQuery: "Toolkit override"},
			cfgDesc:      strPtr("Per-reg override"),
			toolName:     ToolQuery,
			wantDesc:     "Per-reg override",
		},
		{
			name:         "toolkit override for one tool, default for another",
			toolkitDescs: map[ToolName]string{ToolQuery: "Custom query desc"},
			cfgDesc:      nil,
			toolName:     ToolExplain,
			wantDesc:     defaultDescriptions[ToolExplain],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &Toolkit{
				descriptions: make(map[ToolName]string),
			}
			if tt.toolkitDescs != nil {
				tk.descriptions = tt.toolkitDescs
			}

			var cfg *toolConfig
			if tt.cfgDesc != nil {
				cfg = &toolConfig{description: tt.cfgDesc}
			}

			got := tk.getDescription(tt.toolName, cfg)
			if got != tt.wantDesc {
				t.Errorf("getDescription() = %q, want %q", got, tt.wantDesc)
			}
		})
	}
}

func TestGetDescription_NilConfig(t *testing.T) {
	tk := &Toolkit{
		descriptions: make(map[ToolName]string),
	}

	got := tk.getDescription(ToolQuery, nil)
	if got != defaultDescriptions[ToolQuery] {
		t.Errorf("expected default description, got %q", got)
	}
}

func strPtr(s string) *string {
	return &s
}
