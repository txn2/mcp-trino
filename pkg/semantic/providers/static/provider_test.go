//nolint:errcheck // test file intentionally ignores some return values
package static

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/txn2/mcp-trino/pkg/semantic"
)

const testYAML = `
tables:
  - catalog: hive
    schema: analytics
    table: users
    description: "User accounts table"
    owners:
      - name: "Data Platform Team"
        type: group
        role: "Technical Owner"
      - id: "user123"
        name: "Alice"
        type: user
    tags:
      - name: pii
        description: "Contains PII"
      - name: daily-refresh
    glossary_terms:
      - "urn:li:glossaryTerm:customer"
    domain:
      urn: "urn:li:domain:customer"
      name: "Customer Analytics"
      description: "Customer data domain"
    columns:
      user_id:
        description: "Unique user identifier"
        tags: [pii]
        sensitive: true
        sensitivity_level: high
      email:
        description: "User email address"
        tags: [pii]
        glossary_terms:
          - "urn:li:glossaryTerm:email"
  - catalog: hive
    schema: analytics
    table: deprecated_table
    description: "Old table"
    deprecated: true
    deprecation_note: "Use new_table instead"
    replaced_by: "new_table"
  - catalog: memory
    schema: default
    table: test
    description: "Test table"
    tags:
      - name: test
    custom_properties:
      created_by: "test"
glossary:
  - urn: "urn:li:glossaryTerm:customer"
    name: "Customer"
    definition: "A person who purchases goods or services"
    related_terms:
      - "urn:li:glossaryTerm:user"
  - urn: "urn:li:glossaryTerm:email"
    name: "Email"
    definition: "Electronic mail address"
`

const testJSON = `{
  "tables": [
    {
      "catalog": "json_catalog",
      "schema": "json_schema",
      "table": "json_table",
      "description": "JSON table"
    }
  ],
  "glossary": []
}`

func createTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

func TestNew(t *testing.T) {
	t.Run("success with YAML file", func(t *testing.T) {
		path := createTestFile(t, "test.yaml", testYAML)
		p, err := New(Config{FilePath: path})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer func() { _ = p.Close() }()

		if p.TableCount() != 3 {
			t.Errorf("TableCount() = %d, want 3", p.TableCount())
		}
		if p.GlossaryCount() != 2 {
			t.Errorf("GlossaryCount() = %d, want 2", p.GlossaryCount())
		}
	})

	t.Run("success with JSON file", func(t *testing.T) {
		path := createTestFile(t, "test.json", testJSON)
		p, err := New(Config{FilePath: path})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer func() { _ = p.Close() }()

		if p.TableCount() != 1 {
			t.Errorf("TableCount() = %d, want 1", p.TableCount())
		}
	})

	t.Run("error on missing file path", func(t *testing.T) {
		_, err := New(Config{})
		if !errors.Is(err, ErrNoFilePath) {
			t.Errorf("New() error = %v, want ErrNoFilePath", err)
		}
	})

	t.Run("error on file not found", func(t *testing.T) {
		_, err := New(Config{FilePath: "/nonexistent/file.yaml"})
		if !errors.Is(err, ErrFileNotFound) {
			t.Errorf("New() error = %v, want ErrFileNotFound", err)
		}
	})

	t.Run("error on unsupported format", func(t *testing.T) {
		path := createTestFile(t, "test.txt", "invalid")
		_, err := New(Config{FilePath: path})
		if !errors.Is(err, ErrUnsupportedFormat) {
			t.Errorf("New() error = %v, want ErrUnsupportedFormat", err)
		}
	})

	t.Run("error on invalid YAML", func(t *testing.T) {
		path := createTestFile(t, "invalid.yaml", "invalid: [yaml: content")
		_, err := New(Config{FilePath: path})
		if err == nil {
			t.Error("New() expected error for invalid YAML")
		}
	})

	t.Run("error on invalid JSON", func(t *testing.T) {
		path := createTestFile(t, "invalid.json", "invalid json")
		_, err := New(Config{FilePath: path})
		if err == nil {
			t.Error("New() expected error for invalid JSON")
		}
	})
}

func TestProvider_Name(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	if got := p.Name(); got != "static" {
		t.Errorf("Name() = %q, want %q", got, "static")
	}
}

func TestProvider_GetTableContext_Basic(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	result, err := p.GetTableContext(ctx, table)
	if err != nil {
		t.Errorf("GetTableContext() error = %v", err)
	}
	if result == nil {
		t.Fatal("GetTableContext() returned nil")
	}
	if result.Description != "User accounts table" {
		t.Errorf("Description = %q, want %q", result.Description, "User accounts table")
	}
	if result.Source != "static" {
		t.Errorf("Source = %q, want %q", result.Source, "static")
	}
	if result.FetchedAt.IsZero() {
		t.Error("FetchedAt should be set")
	}
}

func TestProvider_GetTableContext_Ownership(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	result, _ := p.GetTableContext(ctx, table)
	if result.Ownership == nil {
		t.Fatal("Ownership is nil")
	}
	if len(result.Ownership.Owners) != 2 {
		t.Errorf("Owners count = %d, want 2", len(result.Ownership.Owners))
	}
	if result.Ownership.Owners[0].Name != "Data Platform Team" {
		t.Errorf("Owner[0].Name = %q", result.Ownership.Owners[0].Name)
	}
}

func TestProvider_GetTableContext_Metadata(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	result, _ := p.GetTableContext(ctx, table)

	// Check tags
	if len(result.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(result.Tags))
	}

	// Check glossary terms
	if len(result.GlossaryTerms) != 1 {
		t.Errorf("GlossaryTerms count = %d, want 1", len(result.GlossaryTerms))
	}
	if result.GlossaryTerms[0].URN != "urn:li:glossaryTerm:customer" {
		t.Errorf("GlossaryTerms[0].URN = %q", result.GlossaryTerms[0].URN)
	}

	// Check domain
	if result.Domain == nil {
		t.Fatal("Domain is nil")
	}
	if result.Domain.Name != "Customer Analytics" {
		t.Errorf("Domain.Name = %q", result.Domain.Name)
	}
}

func TestProvider_GetTableContext_Deprecation(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "deprecated_table"}
	result, _ := p.GetTableContext(ctx, table)
	if result.Deprecation == nil {
		t.Fatal("Deprecation is nil")
	}
	if !result.Deprecation.Deprecated {
		t.Error("Deprecated should be true")
	}
	if result.Deprecation.ReplacedBy != "new_table" {
		t.Errorf("ReplacedBy = %q", result.Deprecation.ReplacedBy)
	}
}

func TestProvider_GetTableContext_CustomProperties(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "memory", Schema: "default", Table: "test"}
	result, _ := p.GetTableContext(ctx, table)
	if result.CustomProperties == nil {
		t.Fatal("CustomProperties is nil")
	}
	if result.CustomProperties["created_by"] != "test" {
		t.Errorf("CustomProperties[created_by] = %v", result.CustomProperties["created_by"])
	}
}

func TestProvider_GetTableContext_Unknown(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "unknown", Schema: "unknown", Table: "unknown"}
	result, err := p.GetTableContext(ctx, table)
	if err != nil {
		t.Errorf("GetTableContext() error = %v", err)
	}
	if result != nil {
		t.Errorf("GetTableContext() = %v, want nil", result)
	}
}

func TestProvider_GetColumnContext(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	t.Run("returns column context", func(t *testing.T) {
		column := semantic.ColumnIdentifier{
			TableIdentifier: semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
			Column:          "user_id",
		}
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result == nil {
			t.Fatal("GetColumnContext() returned nil")
		}
		if result.Description != "Unique user identifier" {
			t.Errorf("Description = %q", result.Description)
		}
		if !result.IsSensitive {
			t.Error("IsSensitive should be true")
		}
		if result.SensitivityLevel != "high" {
			t.Errorf("SensitivityLevel = %q", result.SensitivityLevel)
		}
	})

	t.Run("includes tags", func(t *testing.T) {
		column := semantic.ColumnIdentifier{
			TableIdentifier: semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
			Column:          "user_id",
		}
		result, _ := p.GetColumnContext(ctx, column)
		if len(result.Tags) != 1 {
			t.Errorf("Tags count = %d, want 1", len(result.Tags))
		}
		if result.Tags[0].Name != "pii" {
			t.Errorf("Tags[0].Name = %q", result.Tags[0].Name)
		}
	})

	t.Run("includes glossary terms", func(t *testing.T) {
		column := semantic.ColumnIdentifier{
			TableIdentifier: semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
			Column:          "email",
		}
		result, _ := p.GetColumnContext(ctx, column)
		if len(result.GlossaryTerms) != 1 {
			t.Errorf("GlossaryTerms count = %d, want 1", len(result.GlossaryTerms))
		}
	})

	t.Run("returns nil for unknown table", func(t *testing.T) {
		column := semantic.ColumnIdentifier{
			TableIdentifier: semantic.TableIdentifier{Catalog: "unknown", Schema: "unknown", Table: "unknown"},
			Column:          "col",
		}
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnContext() = %v, want nil", result)
		}
	})

	t.Run("returns nil for unknown column", func(t *testing.T) {
		column := semantic.ColumnIdentifier{
			TableIdentifier: semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
			Column:          "unknown",
		}
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnContext() = %v, want nil", result)
		}
	})
}

func TestProvider_GetColumnsContext(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	t.Run("returns all columns", func(t *testing.T) {
		table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
		result, err := p.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
		if result["user_id"] == nil {
			t.Error("user_id column missing")
		}
		if result["email"] == nil {
			t.Error("email column missing")
		}
	})

	t.Run("returns nil for unknown table", func(t *testing.T) {
		table := semantic.TableIdentifier{Catalog: "unknown", Schema: "unknown", Table: "unknown"}
		result, err := p.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnsContext() = %v, want nil", result)
		}
	})

	t.Run("returns nil for table without columns", func(t *testing.T) {
		table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "deprecated_table"}
		result, err := p.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnsContext() = %v, want nil", result)
		}
	})
}

func TestProvider_GetLineage(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}

	result, err := p.GetLineage(ctx, table, semantic.LineageUpstream, 3)
	if err != nil {
		t.Errorf("GetLineage() error = %v", err)
	}
	if result != nil {
		t.Errorf("GetLineage() = %v, want nil (not supported)", result)
	}
}

func TestProvider_GetGlossaryTerm(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	t.Run("returns glossary term", func(t *testing.T) {
		result, err := p.GetGlossaryTerm(ctx, "urn:li:glossaryTerm:customer")
		if err != nil {
			t.Errorf("GetGlossaryTerm() error = %v", err)
		}
		if result == nil {
			t.Fatal("GetGlossaryTerm() returned nil")
		}
		if result.Name != "Customer" {
			t.Errorf("Name = %q", result.Name)
		}
		if result.Definition != "A person who purchases goods or services" {
			t.Errorf("Definition = %q", result.Definition)
		}
		if len(result.RelatedTerms) != 1 {
			t.Errorf("RelatedTerms count = %d, want 1", len(result.RelatedTerms))
		}
		if result.Source != "static" {
			t.Errorf("Source = %q", result.Source)
		}
	})

	t.Run("returns nil for unknown term", func(t *testing.T) {
		result, err := p.GetGlossaryTerm(ctx, "urn:li:glossaryTerm:unknown")
		if err != nil {
			t.Errorf("GetGlossaryTerm() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetGlossaryTerm() = %v, want nil", result)
		}
	})
}

func TestProvider_SearchTables(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()

	tests := []struct {
		name      string
		filter    semantic.SearchFilter
		wantCount int
	}{
		{"all tables with deprecated", semantic.SearchFilter{IncludeDeprecated: true}, 3},
		{"excludes deprecated by default", semantic.SearchFilter{}, 2},
		{"filter by catalog", semantic.SearchFilter{Catalog: "memory", IncludeDeprecated: true}, 1},
		{"filter by schema", semantic.SearchFilter{Schema: "default", IncludeDeprecated: true}, 1},
		{"filter by domain name", semantic.SearchFilter{Domain: "Customer Analytics"}, 1},
		{"filter by domain URN", semantic.SearchFilter{Domain: "urn:li:domain:customer"}, 1},
		{"filter by owner ID", semantic.SearchFilter{Owner: "user123"}, 1},
		{"filter by owner name", semantic.SearchFilter{Owner: "Alice"}, 1},
		{"filter by tags", semantic.SearchFilter{Tags: []string{"pii"}}, 1},
		{"filter by multiple tags", semantic.SearchFilter{Tags: []string{"pii", "daily-refresh"}}, 1},
		{"filter by query in table name", semantic.SearchFilter{Query: "users", IncludeDeprecated: true}, 1},
		{"filter by query in description", semantic.SearchFilter{Query: "accounts", IncludeDeprecated: true}, 1},
		{"applies limit", semantic.SearchFilter{Limit: 1, IncludeDeprecated: true}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.SearchTables(ctx, tt.filter)
			if err != nil {
				t.Errorf("SearchTables() error = %v", err)
			}
			if len(result) != tt.wantCount {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestProvider_SearchTables_CatalogMatch(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	result, err := p.SearchTables(ctx, semantic.SearchFilter{Catalog: "memory", IncludeDeprecated: true})
	if err != nil {
		t.Errorf("SearchTables() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].Catalog != "memory" {
		t.Errorf("result[0].Catalog = %q, want memory", result[0].Catalog)
	}
}

func TestProvider_Close(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = p.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Close should be idempotent
	err = p.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestProvider_Reload(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	initialCount := p.TableCount()

	// Modify the file
	newContent := `
tables:
  - catalog: new
    schema: new
    table: new
glossary: []
`
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	// Reload
	if err := p.Reload(); err != nil {
		t.Errorf("Reload() error = %v", err)
	}

	if p.TableCount() != 1 {
		t.Errorf("TableCount() = %d, want 1 after reload", p.TableCount())
	}
	if p.TableCount() == initialCount {
		t.Error("Reload() did not update data")
	}
}

func TestProvider_WatchInterval(t *testing.T) {
	path := createTestFile(t, "test.yaml", testYAML)
	p, err := New(Config{
		FilePath:      path,
		WatchInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = p.Close() }()

	// Modify the file
	newContent := `
tables:
  - catalog: watched
    schema: watched
    table: watched
glossary: []
`
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	// Wait for watcher to pick up changes
	time.Sleep(100 * time.Millisecond)

	if p.TableCount() != 1 {
		t.Errorf("TableCount() = %d, want 1 after watch", p.TableCount())
	}

	ctx := context.Background()
	table := semantic.TableIdentifier{Catalog: "watched", Schema: "watched", Table: "watched"}
	result, _ := p.GetTableContext(ctx, table)
	if result == nil {
		t.Error("watched table not found after auto-reload")
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := Config{FilePath: "/some/path.yaml"}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v", err)
		}
	})

	t.Run("empty file path", func(t *testing.T) {
		cfg := Config{}
		if err := cfg.Validate(); !errors.Is(err, ErrNoFilePath) {
			t.Errorf("Validate() error = %v, want ErrNoFilePath", err)
		}
	})
}

func TestFromEnv(t *testing.T) {
	// Save and restore env
	oldFile := os.Getenv("SEMANTIC_STATIC_FILE")
	oldInterval := os.Getenv("SEMANTIC_STATIC_WATCH_INTERVAL")
	defer func() {
		_ = os.Setenv("SEMANTIC_STATIC_FILE", oldFile)
		_ = os.Setenv("SEMANTIC_STATIC_WATCH_INTERVAL", oldInterval)
	}()

	t.Run("reads file path", func(t *testing.T) {
		t.Setenv("SEMANTIC_STATIC_FILE", "/test/path.yaml")
		t.Setenv("SEMANTIC_STATIC_WATCH_INTERVAL", "")
		cfg := FromEnv()
		if cfg.FilePath != "/test/path.yaml" {
			t.Errorf("FilePath = %q", cfg.FilePath)
		}
	})

	t.Run("reads watch interval", func(t *testing.T) {
		t.Setenv("SEMANTIC_STATIC_FILE", "/test/path.yaml")
		t.Setenv("SEMANTIC_STATIC_WATCH_INTERVAL", "30s")
		cfg := FromEnv()
		if cfg.WatchInterval != 30*time.Second {
			t.Errorf("WatchInterval = %v", cfg.WatchInterval)
		}
	})

	t.Run("ignores invalid interval", func(t *testing.T) {
		t.Setenv("SEMANTIC_STATIC_FILE", "/test/path.yaml")
		t.Setenv("SEMANTIC_STATIC_WATCH_INTERVAL", "invalid")
		cfg := FromEnv()
		if cfg.WatchInterval != 0 {
			t.Errorf("WatchInterval = %v, want 0", cfg.WatchInterval)
		}
	})
}

func TestProvider_ImplementsInterface(_ *testing.T) {
	var _ semantic.Provider = (*Provider)(nil)
}
