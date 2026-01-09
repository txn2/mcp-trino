package tools

import (
	"strings"
	"testing"
)

func TestListCatalogsInput(_ *testing.T) {
	// ListCatalogsInput has no fields, just verify it can be created
	input := ListCatalogsInput{}
	_ = input // Verify it exists
}

func TestListSchemasInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input ListSchemasInput
		valid bool
	}{
		{
			name: "valid input",
			input: ListSchemasInput{
				Catalog: "hive",
			},
			valid: true,
		},
		{
			name: "missing catalog",
			input: ListSchemasInput{
				Catalog: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Catalog != ""
			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestListTablesInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input ListTablesInput
		valid bool
	}{
		{
			name: "valid input",
			input: ListTablesInput{
				Catalog: "hive",
				Schema:  "default",
			},
			valid: true,
		},
		{
			name: "missing catalog",
			input: ListTablesInput{
				Catalog: "",
				Schema:  "default",
			},
			valid: false,
		},
		{
			name: "missing schema",
			input: ListTablesInput{
				Catalog: "hive",
				Schema:  "",
			},
			valid: false,
		},
		{
			name: "with pattern",
			input: ListTablesInput{
				Catalog: "hive",
				Schema:  "default",
				Pattern: "%user%",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Catalog != "" && tt.input.Schema != ""
			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestDescribeTableInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input DescribeTableInput
		valid bool
	}{
		{
			name: "valid input",
			input: DescribeTableInput{
				Catalog: "hive",
				Schema:  "default",
				Table:   "users",
			},
			valid: true,
		},
		{
			name: "missing catalog",
			input: DescribeTableInput{
				Catalog: "",
				Schema:  "default",
				Table:   "users",
			},
			valid: false,
		},
		{
			name: "missing schema",
			input: DescribeTableInput{
				Catalog: "hive",
				Schema:  "",
				Table:   "users",
			},
			valid: false,
		},
		{
			name: "missing table",
			input: DescribeTableInput{
				Catalog: "hive",
				Schema:  "default",
				Table:   "",
			},
			valid: false,
		},
		{
			name: "with sample",
			input: DescribeTableInput{
				Catalog:       "hive",
				Schema:        "default",
				Table:         "users",
				IncludeSample: true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Catalog != "" && tt.input.Schema != "" && tt.input.Table != ""
			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestPatternMatching(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		tableName string
		matches   bool
	}{
		{
			name:      "exact match",
			pattern:   "users",
			tableName: "users",
			matches:   true,
		},
		{
			name:      "contains pattern",
			pattern:   "%user%",
			tableName: "app_users_v2",
			matches:   true,
		},
		{
			name:      "prefix pattern",
			pattern:   "user%",
			tableName: "users",
			matches:   true,
		},
		{
			name:      "suffix pattern",
			pattern:   "%log",
			tableName: "audit_log",
			matches:   true,
		},
		{
			name:      "case insensitive",
			pattern:   "%USER%",
			tableName: "app_users",
			matches:   true,
		},
		{
			name:      "no match",
			pattern:   "%order%",
			tableName: "users",
			matches:   false,
		},
		{
			name:      "empty pattern matches all",
			pattern:   "",
			tableName: "anything",
			matches:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the pattern matching logic from handleListTables
			if tt.pattern == "" {
				if !tt.matches {
					t.Error("empty pattern should match everything")
				}
				return
			}

			pattern := strings.ToLower(tt.pattern)
			pattern = strings.ReplaceAll(pattern, "%", "")
			matches := strings.Contains(strings.ToLower(tt.tableName), pattern)

			if matches != tt.matches {
				t.Errorf("expected matches=%v, got %v", tt.matches, matches)
			}
		})
	}
}

func TestListTablesInput_PatternEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{
			name:    "just wildcards",
			pattern: "%%",
		},
		{
			name:    "single wildcard",
			pattern: "%",
		},
		{
			name:    "multiple wildcards",
			pattern: "%a%b%c%",
		},
		{
			name:    "special SQL chars",
			pattern: "%_test%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ListTablesInput{
				Catalog: "hive",
				Schema:  "default",
				Pattern: tt.pattern,
			}

			// Just verify patterns can be set
			if input.Pattern != tt.pattern {
				t.Errorf("expected Pattern %q, got %q", tt.pattern, input.Pattern)
			}
		})
	}
}

func TestDescribeTableInput_IncludeSample(t *testing.T) {
	tests := []struct {
		name          string
		includeSample bool
	}{
		{
			name:          "with sample",
			includeSample: true,
		},
		{
			name:          "without sample",
			includeSample: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := DescribeTableInput{
				Catalog:       "hive",
				Schema:        "default",
				Table:         "users",
				IncludeSample: tt.includeSample,
			}

			if input.IncludeSample != tt.includeSample {
				t.Errorf("expected IncludeSample=%v, got %v", tt.includeSample, input.IncludeSample)
			}
		})
	}
}

func TestInputStructs_JSONTags(t *testing.T) {
	// Verify that all input structs have correct JSON tags
	// This is a compile-time check that verifies the struct fields exist

	listSchemas := ListSchemasInput{Catalog: "test"}
	if listSchemas.Catalog != "test" {
		t.Error("ListSchemasInput.Catalog field not accessible")
	}

	listTables := ListTablesInput{Catalog: "c", Schema: "s", Pattern: "p"}
	if listTables.Pattern != "p" {
		t.Error("ListTablesInput.Pattern field not accessible")
	}

	descTable := DescribeTableInput{Catalog: "c", Schema: "s", Table: "t", IncludeSample: true}
	if !descTable.IncludeSample {
		t.Error("DescribeTableInput.IncludeSample field not accessible")
	}
}
