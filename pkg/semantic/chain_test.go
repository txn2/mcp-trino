package semantic

import (
	"context"
	"errors"
	"testing"
)

func TestProviderChain_Name(t *testing.T) {
	tests := []struct {
		name      string
		providers []Provider
		expected  string
	}{
		{
			name:      "empty chain",
			providers: nil,
			expected:  "chain(empty)",
		},
		{
			name:      "single provider",
			providers: []Provider{ProviderFunc{NameFn: func() string { return "single" }}},
			expected:  "single",
		},
		{
			name: "multiple providers",
			providers: []Provider{
				ProviderFunc{NameFn: func() string { return "first" }},
				ProviderFunc{NameFn: func() string { return "second" }},
			},
			expected: "chain(first,second)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewProviderChain(tt.providers...)
			got := chain.Name()
			if got != tt.expected {
				t.Errorf("Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProviderChain_GetTableContext(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	t.Run("first provider returns result", func(t *testing.T) {
		expected := &TableContext{Description: "from first"}
		chain := NewProviderChain(
			ProviderFunc{
				GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
					return expected, nil
				},
			},
			ProviderFunc{
				GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
					t.Error("second provider should not be called")
					return nil, nil
				},
			},
		)

		result, err := chain.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("GetTableContext() error = %v", err)
		}
		if result != expected {
			t.Errorf("GetTableContext() = %v, want %v", result, expected)
		}
	})

	t.Run("falls through to second provider", func(t *testing.T) {
		expected := &TableContext{Description: "from second"}
		chain := NewProviderChain(
			ProviderFunc{
				GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
					return nil, nil // No result
				},
			},
			ProviderFunc{
				GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
					return expected, nil
				},
			},
		)

		result, err := chain.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("GetTableContext() error = %v", err)
		}
		if result != expected {
			t.Errorf("GetTableContext() = %v, want %v", result, expected)
		}
	})

	t.Run("error stops chain", func(t *testing.T) {
		expectedErr := errors.New("provider error")
		chain := NewProviderChain(
			ProviderFunc{
				GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
					return nil, expectedErr
				},
			},
			ProviderFunc{
				GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
					t.Error("second provider should not be called after error")
					return nil, nil
				},
			},
		)

		_, err := chain.GetTableContext(ctx, table)
		if !errors.Is(err, expectedErr) {
			t.Errorf("GetTableContext() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("all providers return nil", func(t *testing.T) {
		chain := NewProviderChain(
			ProviderFunc{},
			ProviderFunc{},
		)

		result, err := chain.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("GetTableContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetTableContext() = %v, want nil", result)
		}
	})
}

func TestProviderChain_GetColumnContext(t *testing.T) {
	ctx := context.Background()
	column := ColumnIdentifier{
		TableIdentifier: TableIdentifier{Catalog: "test", Schema: "test", Table: "test"},
		Column:          "col1",
	}

	expected := &ColumnContext{Description: "test column"}
	chain := NewProviderChain(
		ProviderFunc{},
		ProviderFunc{
			GetColumnContextFn: func(_ context.Context, _ ColumnIdentifier) (*ColumnContext, error) {
				return expected, nil
			},
		},
	)

	result, err := chain.GetColumnContext(ctx, column)
	if err != nil {
		t.Errorf("GetColumnContext() error = %v", err)
	}
	if result != expected {
		t.Errorf("GetColumnContext() = %v, want %v", result, expected)
	}
}

func TestProviderChain_GetColumnsContext(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	t.Run("merges results from multiple providers", func(t *testing.T) {
		chain := NewProviderChain(
			ProviderFunc{
				GetColumnsContextFn: func(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
					return map[string]*ColumnContext{
						"col1": {Description: "from first"},
					}, nil
				},
			},
			ProviderFunc{
				GetColumnsContextFn: func(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
					return map[string]*ColumnContext{
						"col2": {Description: "from second"},
					}, nil
				},
			},
		)

		result, err := chain.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("GetColumnsContext() len = %d, want 2", len(result))
		}
		if result["col1"].Description != "from first" {
			t.Errorf("col1 description = %q, want %q", result["col1"].Description, "from first")
		}
		if result["col2"].Description != "from second" {
			t.Errorf("col2 description = %q, want %q", result["col2"].Description, "from second")
		}
	})

	t.Run("later providers override earlier ones", func(t *testing.T) {
		chain := NewProviderChain(
			ProviderFunc{
				GetColumnsContextFn: func(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
					return map[string]*ColumnContext{
						"col1": {Description: "from first"},
					}, nil
				},
			},
			ProviderFunc{
				GetColumnsContextFn: func(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
					return map[string]*ColumnContext{
						"col1": {Description: "from second"},
					}, nil
				},
			},
		)

		result, err := chain.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if result["col1"].Description != "from second" {
			t.Errorf("col1 description = %q, want %q", result["col1"].Description, "from second")
		}
	})

	t.Run("returns nil when all providers return nil", func(t *testing.T) {
		chain := NewProviderChain(ProviderFunc{}, ProviderFunc{})
		result, err := chain.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnsContext() = %v, want nil", result)
		}
	})
}

func TestProviderChain_GetLineage(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	expected := &LineageInfo{Direction: LineageUpstream}
	chain := NewProviderChain(
		ProviderFunc{},
		ProviderFunc{
			GetLineageFn: func(_ context.Context, _ TableIdentifier, _ LineageDirection, _ int) (*LineageInfo, error) {
				return expected, nil
			},
		},
	)

	result, err := chain.GetLineage(ctx, table, LineageUpstream, 3)
	if err != nil {
		t.Errorf("GetLineage() error = %v", err)
	}
	if result != expected {
		t.Errorf("GetLineage() = %v, want %v", result, expected)
	}
}

func TestProviderChain_GetGlossaryTerm(t *testing.T) {
	ctx := context.Background()
	urn := "urn:li:glossaryTerm:test"

	expected := &GlossaryTerm{Name: "Test"}
	chain := NewProviderChain(
		ProviderFunc{},
		ProviderFunc{
			GetGlossaryTermFn: func(_ context.Context, _ string) (*GlossaryTerm, error) {
				return expected, nil
			},
		},
	)

	result, err := chain.GetGlossaryTerm(ctx, urn)
	if err != nil {
		t.Errorf("GetGlossaryTerm() error = %v", err)
	}
	if result != expected {
		t.Errorf("GetGlossaryTerm() = %v, want %v", result, expected)
	}
}

func TestProviderChain_SearchTables(t *testing.T) {
	ctx := context.Background()
	filter := SearchFilter{Query: "test"}

	t.Run("combines results from multiple providers", func(t *testing.T) {
		chain := NewProviderChain(
			ProviderFunc{
				SearchTablesFn: func(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
					return []TableIdentifier{{Catalog: "c1", Schema: "s1", Table: "t1"}}, nil
				},
			},
			ProviderFunc{
				SearchTablesFn: func(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
					return []TableIdentifier{{Catalog: "c2", Schema: "s2", Table: "t2"}}, nil
				},
			},
		)

		result, err := chain.SearchTables(ctx, filter)
		if err != nil {
			t.Errorf("SearchTables() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("SearchTables() len = %d, want 2", len(result))
		}
	})

	t.Run("deduplicates results", func(t *testing.T) {
		chain := NewProviderChain(
			ProviderFunc{
				SearchTablesFn: func(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
					return []TableIdentifier{{Catalog: "c1", Schema: "s1", Table: "t1"}}, nil
				},
			},
			ProviderFunc{
				SearchTablesFn: func(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
					return []TableIdentifier{{Catalog: "c1", Schema: "s1", Table: "t1"}}, nil
				},
			},
		)

		result, err := chain.SearchTables(ctx, filter)
		if err != nil {
			t.Errorf("SearchTables() error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("SearchTables() len = %d, want 1 (deduplicated)", len(result))
		}
	})
}

func TestProviderChain_Close(t *testing.T) {
	t.Run("closes all providers", func(t *testing.T) {
		closed1, closed2 := false, false
		chain := NewProviderChain(
			ProviderFunc{CloseFn: func() error { closed1 = true; return nil }},
			ProviderFunc{CloseFn: func() error { closed2 = true; return nil }},
		)

		err := chain.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
		if !closed1 {
			t.Error("first provider not closed")
		}
		if !closed2 {
			t.Error("second provider not closed")
		}
	})

	t.Run("returns first error", func(t *testing.T) {
		expectedErr := errors.New("close error")
		chain := NewProviderChain(
			ProviderFunc{CloseFn: func() error { return expectedErr }},
			ProviderFunc{CloseFn: func() error { return errors.New("second error") }},
		)

		err := chain.Close()
		if !errors.Is(err, expectedErr) {
			t.Errorf("Close() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestProviderChain_Append(t *testing.T) {
	chain := NewProviderChain(ProviderFunc{NameFn: func() string { return "first" }})
	if chain.Len() != 1 {
		t.Errorf("initial Len() = %d, want 1", chain.Len())
	}

	chain.Append(ProviderFunc{NameFn: func() string { return "second" }})
	if chain.Len() != 2 {
		t.Errorf("after Append() Len() = %d, want 2", chain.Len())
	}
}

func TestProviderChain_ImplementsInterface(_ *testing.T) {
	var _ Provider = (*ProviderChain)(nil)
}
