package tools

import (
	"testing"
)

func TestToolName_String(t *testing.T) {
	tests := []struct {
		name     ToolName
		expected string
	}{
		{ToolQuery, "trino_query"},
		{ToolExecute, "trino_execute"},
		{ToolExplain, "trino_explain"},
		{ToolListCatalogs, "trino_list_catalogs"},
		{ToolListSchemas, "trino_list_schemas"},
		{ToolListTables, "trino_list_tables"},
		{ToolDescribeTable, "trino_describe_table"},
		{ToolListConnections, "trino_list_connections"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			if tt.name.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.name.String())
			}
		})
	}
}

func TestAllTools(t *testing.T) {
	tools := AllTools()

	if len(tools) != 8 {
		t.Errorf("expected 8 tools, got %d", len(tools))
	}

	// Verify all expected tools are present
	expected := map[ToolName]bool{
		ToolQuery:           false,
		ToolExecute:         false,
		ToolExplain:         false,
		ToolListCatalogs:    false,
		ToolListSchemas:     false,
		ToolListTables:      false,
		ToolDescribeTable:   false,
		ToolListConnections: false,
	}

	for _, tool := range tools {
		if _, ok := expected[tool]; !ok {
			t.Errorf("unexpected tool: %v", tool)
		}
		expected[tool] = true
	}

	for tool, found := range expected {
		if !found {
			t.Errorf("missing tool: %v", tool)
		}
	}
}

func TestQueryTools(t *testing.T) {
	tools := QueryTools()

	if len(tools) != 3 {
		t.Errorf("expected 3 query tools, got %d", len(tools))
	}

	// Should contain ToolQuery, ToolExecute, and ToolExplain
	hasQuery := false
	hasExecute := false
	hasExplain := false

	for _, tool := range tools {
		switch tool {
		case ToolQuery:
			hasQuery = true
		case ToolExecute:
			hasExecute = true
		case ToolExplain:
			hasExplain = true
		default:
			t.Errorf("unexpected tool in QueryTools: %v", tool)
		}
	}

	if !hasQuery {
		t.Error("missing ToolQuery")
	}
	if !hasExecute {
		t.Error("missing ToolExecute")
	}
	if !hasExplain {
		t.Error("missing ToolExplain")
	}
}

func TestSchemaTools(t *testing.T) {
	tools := SchemaTools()

	if len(tools) != 4 {
		t.Errorf("expected 4 schema tools, got %d", len(tools))
	}

	expected := map[ToolName]bool{
		ToolListCatalogs:  false,
		ToolListSchemas:   false,
		ToolListTables:    false,
		ToolDescribeTable: false,
	}

	for _, tool := range tools {
		if _, ok := expected[tool]; !ok {
			t.Errorf("unexpected tool in SchemaTools: %v", tool)
		}
		expected[tool] = true
	}

	for tool, found := range expected {
		if !found {
			t.Errorf("missing tool: %v", tool)
		}
	}
}
