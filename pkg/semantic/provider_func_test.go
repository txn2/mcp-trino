package semantic

import (
	"context"
	"errors"
	"testing"
)

func TestProviderFunc_Name(t *testing.T) {
	tests := []struct {
		name     string
		pf       ProviderFunc
		expected string
	}{
		{
			name:     "with NameFn",
			pf:       ProviderFunc{NameFn: func() string { return "custom" }},
			expected: "custom",
		},
		{
			name:     "without NameFn",
			pf:       ProviderFunc{},
			expected: "func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pf.Name()
			if got != tt.expected {
				t.Errorf("Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProviderFunc_GetTableContext(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	t.Run("with function", func(t *testing.T) {
		expected := &TableContext{Description: "test description"}
		pf := ProviderFunc{
			GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
				return expected, nil
			},
		}

		result, err := pf.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("GetTableContext() error = %v", err)
		}
		if result != expected {
			t.Errorf("GetTableContext() = %v, want %v", result, expected)
		}
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		pf := ProviderFunc{
			GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
				return nil, expectedErr
			},
		}

		_, err := pf.GetTableContext(ctx, table)
		if !errors.Is(err, expectedErr) {
			t.Errorf("GetTableContext() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		result, err := pf.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("GetTableContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetTableContext() = %v, want nil", result)
		}
	})
}

func TestProviderFunc_GetColumnContext(t *testing.T) {
	ctx := context.Background()
	column := ColumnIdentifier{
		TableIdentifier: TableIdentifier{Catalog: "test", Schema: "test", Table: "test"},
		Column:          "col1",
	}

	t.Run("with function", func(t *testing.T) {
		expected := &ColumnContext{Description: "test column"}
		pf := ProviderFunc{
			GetColumnContextFn: func(_ context.Context, _ ColumnIdentifier) (*ColumnContext, error) {
				return expected, nil
			},
		}

		result, err := pf.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result != expected {
			t.Errorf("GetColumnContext() = %v, want %v", result, expected)
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		result, err := pf.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnContext() = %v, want nil", result)
		}
	})
}

func TestProviderFunc_GetColumnsContext(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	t.Run("with function", func(t *testing.T) {
		expected := map[string]*ColumnContext{
			"col1": {Description: "column 1"},
		}
		pf := ProviderFunc{
			GetColumnsContextFn: func(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
				return expected, nil
			},
		}

		result, err := pf.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if len(result) != len(expected) {
			t.Errorf("GetColumnsContext() len = %d, want %d", len(result), len(expected))
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		result, err := pf.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnsContext() = %v, want nil", result)
		}
	})
}

func TestProviderFunc_GetLineage(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	t.Run("with function", func(t *testing.T) {
		expected := &LineageInfo{Direction: LineageUpstream}
		pf := ProviderFunc{
			GetLineageFn: func(_ context.Context, _ TableIdentifier, _ LineageDirection, _ int) (*LineageInfo, error) {
				return expected, nil
			},
		}

		result, err := pf.GetLineage(ctx, table, LineageUpstream, 3)
		if err != nil {
			t.Errorf("GetLineage() error = %v", err)
		}
		if result != expected {
			t.Errorf("GetLineage() = %v, want %v", result, expected)
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		result, err := pf.GetLineage(ctx, table, LineageUpstream, 3)
		if err != nil {
			t.Errorf("GetLineage() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetLineage() = %v, want nil", result)
		}
	})
}

func TestProviderFunc_GetGlossaryTerm(t *testing.T) {
	ctx := context.Background()
	urn := "urn:li:glossaryTerm:test"

	t.Run("with function", func(t *testing.T) {
		expected := &GlossaryTerm{Name: "Test Term"}
		pf := ProviderFunc{
			GetGlossaryTermFn: func(_ context.Context, _ string) (*GlossaryTerm, error) {
				return expected, nil
			},
		}

		result, err := pf.GetGlossaryTerm(ctx, urn)
		if err != nil {
			t.Errorf("GetGlossaryTerm() error = %v", err)
		}
		if result != expected {
			t.Errorf("GetGlossaryTerm() = %v, want %v", result, expected)
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		result, err := pf.GetGlossaryTerm(ctx, urn)
		if err != nil {
			t.Errorf("GetGlossaryTerm() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetGlossaryTerm() = %v, want nil", result)
		}
	})
}

func TestProviderFunc_SearchTables(t *testing.T) {
	ctx := context.Background()
	filter := SearchFilter{Query: "test"}

	t.Run("with function", func(t *testing.T) {
		expected := []TableIdentifier{{Catalog: "test", Schema: "test", Table: "test"}}
		pf := ProviderFunc{
			SearchTablesFn: func(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
				return expected, nil
			},
		}

		result, err := pf.SearchTables(ctx, filter)
		if err != nil {
			t.Errorf("SearchTables() error = %v", err)
		}
		if len(result) != len(expected) {
			t.Errorf("SearchTables() len = %d, want %d", len(result), len(expected))
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		result, err := pf.SearchTables(ctx, filter)
		if err != nil {
			t.Errorf("SearchTables() error = %v", err)
		}
		if result != nil {
			t.Errorf("SearchTables() = %v, want nil", result)
		}
	})
}

func TestProviderFunc_Close(t *testing.T) {
	t.Run("with function", func(t *testing.T) {
		called := false
		pf := ProviderFunc{
			CloseFn: func() error {
				called = true
				return nil
			},
		}

		err := pf.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
		if !called {
			t.Error("Close() did not call CloseFn")
		}
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("close error")
		pf := ProviderFunc{
			CloseFn: func() error { return expectedErr },
		}

		err := pf.Close()
		if !errors.Is(err, expectedErr) {
			t.Errorf("Close() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("without function", func(t *testing.T) {
		pf := ProviderFunc{}
		err := pf.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

func TestProviderFunc_ImplementsInterface(_ *testing.T) {
	var _ Provider = ProviderFunc{}
}
