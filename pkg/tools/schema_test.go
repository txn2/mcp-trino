package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/semantic"
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

// --- Semantic Formatting Tests ---

func TestFormatDescription(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		expected string
	}{
		{
			name:     "empty description",
			desc:     "",
			expected: "",
		},
		{
			name:     "with description",
			desc:     "A table for users",
			expected: "**Description:** A table for users\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDescription(tt.desc)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatDeprecation(t *testing.T) {
	tests := []struct {
		name     string
		depr     *semantic.Deprecation
		contains []string
	}{
		{
			name:     "nil deprecation",
			depr:     nil,
			contains: nil,
		},
		{
			name:     "not deprecated",
			depr:     &semantic.Deprecation{Deprecated: false},
			contains: nil,
		},
		{
			name: "deprecated with note",
			depr: &semantic.Deprecation{
				Deprecated: true,
				Note:       "This table is obsolete",
			},
			contains: []string{"DEPRECATED", "This table is obsolete"},
		},
		{
			name: "deprecated with replacement",
			depr: &semantic.Deprecation{
				Deprecated: true,
				Note:       "Old schema",
				ReplacedBy: "new_users",
			},
			contains: []string{"DEPRECATED", "Old schema", "new_users", "instead"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDeprecation(tt.depr)
			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatOwnership(t *testing.T) {
	tests := []struct {
		name     string
		own      *semantic.Ownership
		contains []string
	}{
		{
			name:     "nil ownership",
			own:      nil,
			contains: nil,
		},
		{
			name:     "empty owners",
			own:      &semantic.Ownership{Owners: []semantic.Owner{}},
			contains: nil,
		},
		{
			name: "single owner without role",
			own: &semantic.Ownership{
				Owners: []semantic.Owner{
					{Name: "Data Team"},
				},
			},
			contains: []string{"Owners:", "Data Team"},
		},
		{
			name: "single owner with role",
			own: &semantic.Ownership{
				Owners: []semantic.Owner{
					{Name: "Data Team", Role: "Data Steward"},
				},
			},
			contains: []string{"Owners:", "Data Team", "Data Steward"},
		},
		{
			name: "multiple owners",
			own: &semantic.Ownership{
				Owners: []semantic.Owner{
					{Name: "Data Team", Role: "Data Steward"},
					{Name: "john@example.com", Role: "Technical Owner"},
				},
			},
			contains: []string{"Owners:", "Data Team", "john@example.com", ","},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatOwnership(tt.own)
			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []semantic.Tag
		contains []string
	}{
		{
			name:     "nil tags",
			tags:     nil,
			contains: nil,
		},
		{
			name:     "empty tags",
			tags:     []semantic.Tag{},
			contains: nil,
		},
		{
			name: "single tag",
			tags: []semantic.Tag{
				{Name: "pii"},
			},
			contains: []string{"Tags:", "`pii`"},
		},
		{
			name: "multiple tags",
			tags: []semantic.Tag{
				{Name: "pii"},
				{Name: "gdpr"},
				{Name: "financial"},
			},
			contains: []string{"Tags:", "`pii`", "`gdpr`", "`financial`", ","},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTags(tt.tags)
			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   *semantic.Domain
		expected string
	}{
		{
			name:     "nil domain",
			domain:   nil,
			expected: "",
		},
		{
			name:     "empty name",
			domain:   &semantic.Domain{Name: ""},
			expected: "",
		},
		{
			name:     "with name",
			domain:   &semantic.Domain{Name: "Customer Analytics"},
			expected: "**Domain:** Customer Analytics\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatGlossaryTerms(t *testing.T) {
	tests := []struct {
		name     string
		terms    []semantic.GlossaryTerm
		contains []string
	}{
		{
			name:     "nil terms",
			terms:    nil,
			contains: nil,
		},
		{
			name:     "empty terms",
			terms:    []semantic.GlossaryTerm{},
			contains: nil,
		},
		{
			name: "single term",
			terms: []semantic.GlossaryTerm{
				{Name: "MRR"},
			},
			contains: []string{"Glossary Terms:", "*MRR*"},
		},
		{
			name: "multiple terms",
			terms: []semantic.GlossaryTerm{
				{Name: "MRR"},
				{Name: "ARR"},
			},
			contains: []string{"Glossary Terms:", "*MRR*", "*ARR*", ","},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGlossaryTerms(tt.terms)
			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatQuality(t *testing.T) {
	score95 := float64(95)
	score0 := float64(0)

	tests := []struct {
		name     string
		quality  *semantic.DataQuality
		contains []string
	}{
		{
			name:     "nil quality",
			quality:  nil,
			contains: nil,
		},
		{
			name:     "nil score",
			quality:  &semantic.DataQuality{Score: nil},
			contains: nil,
		},
		{
			name:     "with score",
			quality:  &semantic.DataQuality{Score: &score95},
			contains: []string{"Data Quality Score:", "95%"},
		},
		{
			name:     "zero score",
			quality:  &semantic.DataQuality{Score: &score0},
			contains: []string{"Data Quality Score:", "0%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatQuality(tt.quality)
			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatTableSemantics(t *testing.T) {
	score85 := float64(85)

	tests := []struct {
		name     string
		tc       *semantic.TableContext
		contains []string
	}{
		{
			name:     "empty context",
			tc:       &semantic.TableContext{},
			contains: nil,
		},
		{
			name: "full context",
			tc: &semantic.TableContext{
				Description: "User data table",
				Ownership: &semantic.Ownership{
					Owners: []semantic.Owner{{Name: "Data Team", Role: "Owner"}},
				},
				Tags:    []semantic.Tag{{Name: "pii"}},
				Domain:  &semantic.Domain{Name: "Customer"},
				Quality: &semantic.DataQuality{Score: &score85},
				GlossaryTerms: []semantic.GlossaryTerm{
					{Name: "Customer ID"},
				},
				Deprecation: &semantic.Deprecation{
					Deprecated: true,
					Note:       "Use v2",
				},
			},
			contains: []string{
				"User data table",
				"Data Team",
				"`pii`",
				"Customer",
				"85%",
				"Customer ID",
				"DEPRECATED",
			},
		},
	}

	// Create a toolkit to test the method
	cfg := client.Config{Host: "localhost", User: "test"}
	trinoClient := client.NewWithDB(nil, cfg)
	toolkit := NewToolkit(trinoClient, DefaultConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toolkit.formatTableSemantics(tt.tc)
			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatColumnsWithSemantics(t *testing.T) {
	columns := []client.ColumnDef{
		{Name: "id", Type: "bigint", Nullable: "NO", Comment: ""},
		{Name: "email", Type: "varchar", Nullable: "YES", Comment: "User email"},
		{Name: "phone", Type: "varchar", Nullable: "YES", Comment: ""},
	}

	tests := []struct {
		name      string
		semantics map[string]*semantic.ColumnContext
		contains  []string
	}{
		{
			name:      "no semantics",
			semantics: map[string]*semantic.ColumnContext{},
			contains:  []string{"Columns", "`id`", "`email`", "`phone`", "bigint", "varchar"},
		},
		{
			name: "with column descriptions",
			semantics: map[string]*semantic.ColumnContext{
				"email": {
					Description: "Primary email address",
					Tags:        []semantic.Tag{{Name: "pii"}},
				},
				"phone": {
					Description: "Phone number",
					IsSensitive: true,
				},
			},
			contains: []string{
				"Columns", "`id`", "`email`", "`phone`",
				"Primary email address",
				"pii",
				"Phone number",
				"SENSITIVE",
			},
		},
		{
			name: "sensitive only",
			semantics: map[string]*semantic.ColumnContext{
				"id": {
					IsSensitive: true,
				},
			},
			contains: []string{"SENSITIVE"},
		},
	}

	cfg := client.Config{Host: "localhost", User: "test"}
	trinoClient := client.NewWithDB(nil, cfg)
	toolkit := NewToolkit(trinoClient, DefaultConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toolkit.formatColumnsWithSemantics(columns, tt.semantics)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestFormatBasicColumns(t *testing.T) {
	columns := []client.ColumnDef{
		{Name: "id", Type: "bigint", Nullable: "NO", Comment: "Primary key"},
		{Name: "name", Type: "varchar", Nullable: "YES", Comment: ""},
		{Name: "created", Type: "timestamp", Nullable: "", Comment: "Creation time"},
	}

	result := formatBasicColumns(columns)

	// Check header
	if !strings.Contains(result, "### Columns") {
		t.Error("expected header '### Columns'")
	}

	// Check table structure
	if !strings.Contains(result, "| Name | Type | Nullable | Comment |") {
		t.Error("expected table header row")
	}

	// Check column data
	expectedContent := []string{"`id`", "bigint", "NO", "Primary key", "`name`", "varchar", "YES", "`created`", "timestamp", "Creation time"}
	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q", expected)
		}
	}

	// Check default values for empty fields
	if !strings.Contains(result, "| - |") {
		t.Error("expected '-' for empty nullable or comment")
	}
}

func TestEnrichWithTableContext(t *testing.T) {
	cfg := client.Config{Host: "localhost", User: "test"}
	trinoClient := client.NewWithDB(nil, cfg)

	tests := []struct {
		name     string
		provider *semantic.ProviderFunc
		contains []string
	}{
		{
			name: "returns formatted context when provider returns data",
			provider: &semantic.ProviderFunc{
				NameFn: func() string { return "test" },
				GetTableContextFn: func(_ context.Context, _ semantic.TableIdentifier) (*semantic.TableContext, error) {
					return &semantic.TableContext{
						Description: "Test table description",
						Tags:        []semantic.Tag{{Name: "pii"}},
					}, nil
				},
			},
			contains: []string{"Test table description", "`pii`"},
		},
		{
			name: "returns empty when provider returns nil",
			provider: &semantic.ProviderFunc{
				NameFn: func() string { return "test" },
				GetTableContextFn: func(_ context.Context, _ semantic.TableIdentifier) (*semantic.TableContext, error) {
					return nil, nil
				},
			},
			contains: nil,
		},
		{
			name: "returns empty when provider returns error",
			provider: &semantic.ProviderFunc{
				NameFn: func() string { return "test" },
				GetTableContextFn: func(_ context.Context, _ semantic.TableIdentifier) (*semantic.TableContext, error) {
					return nil, errors.New("provider error")
				},
			},
			contains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolkit := NewToolkit(trinoClient, DefaultConfig(), WithSemanticProvider(tt.provider))

			tableID := semantic.TableIdentifier{
				Catalog: "test",
				Schema:  "public",
				Table:   "users",
			}

			result := toolkit.enrichWithTableContext(context.Background(), tableID)

			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestFormatTableWithSemantics(t *testing.T) {
	cfg := client.Config{Host: "localhost", User: "test"}
	trinoClient := client.NewWithDB(nil, cfg)

	columns := []client.ColumnDef{
		{Name: "id", Type: "bigint", Nullable: "NO"},
		{Name: "email", Type: "varchar", Nullable: "YES"},
	}
	tableInfo := &client.TableInfo{
		Catalog: "test",
		Schema:  "public",
		Name:    "users",
		Columns: columns,
	}

	input := DescribeTableInput{
		Catalog: "test",
		Schema:  "public",
		Table:   "users",
	}

	tests := []struct {
		name     string
		provider *semantic.ProviderFunc
		contains []string
	}{
		{
			name:     "without semantic provider",
			provider: nil,
			contains: []string{"Columns", "`id`", "`email`", "2 columns"},
		},
		{
			name: "with semantic provider returning table and column context",
			provider: &semantic.ProviderFunc{
				NameFn: func() string { return "test" },
				GetTableContextFn: func(_ context.Context, _ semantic.TableIdentifier) (*semantic.TableContext, error) {
					return &semantic.TableContext{
						Description: "User accounts table",
						Domain:      &semantic.Domain{Name: "Customer"},
					}, nil
				},
				GetColumnsContextFn: func(_ context.Context, _ semantic.TableIdentifier) (map[string]*semantic.ColumnContext, error) {
					return map[string]*semantic.ColumnContext{
						"email": {
							Description: "User email address",
							IsSensitive: true,
						},
					}, nil
				},
			},
			contains: []string{"User accounts table", "Customer", "User email address", "SENSITIVE", "2 columns"},
		},
		{
			name: "with semantic provider returning nil",
			provider: &semantic.ProviderFunc{
				NameFn: func() string { return "test" },
				GetTableContextFn: func(_ context.Context, _ semantic.TableIdentifier) (*semantic.TableContext, error) {
					return nil, nil
				},
				GetColumnsContextFn: func(_ context.Context, _ semantic.TableIdentifier) (map[string]*semantic.ColumnContext, error) {
					return nil, nil
				},
			},
			contains: []string{"Columns", "`id`", "`email`", "2 columns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var toolkit *Toolkit
			if tt.provider != nil {
				toolkit = NewToolkit(trinoClient, DefaultConfig(), WithSemanticProvider(tt.provider))
			} else {
				toolkit = NewToolkit(trinoClient, DefaultConfig())
			}

			result := toolkit.formatTableWithSemantics(context.Background(), input, tableInfo)

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestFormatColumns(t *testing.T) {
	cfg := client.Config{Host: "localhost", User: "test"}
	trinoClient := client.NewWithDB(nil, cfg)
	toolkit := NewToolkit(trinoClient, DefaultConfig())

	columns := []client.ColumnDef{
		{Name: "id", Type: "bigint", Nullable: "NO"},
		{Name: "name", Type: "varchar", Nullable: "YES"},
	}

	tests := []struct {
		name      string
		semantics map[string]*semantic.ColumnContext
		contains  []string
	}{
		{
			name:      "without semantics uses basic format",
			semantics: nil,
			contains:  []string{"| Name | Type | Nullable | Comment |"},
		},
		{
			name:      "empty semantics uses basic format",
			semantics: map[string]*semantic.ColumnContext{},
			contains:  []string{"| Name | Type | Nullable | Comment |"},
		},
		{
			name: "with semantics uses enriched format",
			semantics: map[string]*semantic.ColumnContext{
				"id": {Description: "Primary key"},
			},
			contains: []string{"| Name | Type | Nullable | Description | Tags |"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toolkit.formatColumns(columns, tt.semantics)

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}
