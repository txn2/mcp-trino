package semantic

import (
	"context"
	"testing"
)

func TestNoOpProvider_Name(t *testing.T) {
	p := &NoOpProvider{}
	if got := p.Name(); got != "noop" {
		t.Errorf("Name() = %q, want %q", got, "noop")
	}
}

func TestNoOpProvider_GetTableContext(t *testing.T) {
	p := &NoOpProvider{}
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	result, err := p.GetTableContext(ctx, table)
	if err != nil {
		t.Errorf("GetTableContext() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("GetTableContext() = %v, want nil", result)
	}
}

func TestNoOpProvider_GetColumnContext(t *testing.T) {
	p := &NoOpProvider{}
	ctx := context.Background()
	column := ColumnIdentifier{
		TableIdentifier: TableIdentifier{Catalog: "test", Schema: "test", Table: "test"},
		Column:          "col1",
	}

	result, err := p.GetColumnContext(ctx, column)
	if err != nil {
		t.Errorf("GetColumnContext() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("GetColumnContext() = %v, want nil", result)
	}
}

func TestNoOpProvider_GetColumnsContext(t *testing.T) {
	p := &NoOpProvider{}
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	result, err := p.GetColumnsContext(ctx, table)
	if err != nil {
		t.Errorf("GetColumnsContext() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("GetColumnsContext() = %v, want nil", result)
	}
}

func TestNoOpProvider_GetLineage(t *testing.T) {
	p := &NoOpProvider{}
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	result, err := p.GetLineage(ctx, table, LineageUpstream, 3)
	if err != nil {
		t.Errorf("GetLineage() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("GetLineage() = %v, want nil", result)
	}
}

func TestNoOpProvider_GetGlossaryTerm(t *testing.T) {
	p := &NoOpProvider{}
	ctx := context.Background()

	result, err := p.GetGlossaryTerm(ctx, "urn:li:glossaryTerm:test")
	if err != nil {
		t.Errorf("GetGlossaryTerm() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("GetGlossaryTerm() = %v, want nil", result)
	}
}

func TestNoOpProvider_SearchTables(t *testing.T) {
	p := &NoOpProvider{}
	ctx := context.Background()
	filter := SearchFilter{Query: "test"}

	result, err := p.SearchTables(ctx, filter)
	if err != nil {
		t.Errorf("SearchTables() error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("SearchTables() = %v, want nil", result)
	}
}

func TestNoOpProvider_Close(t *testing.T) {
	p := &NoOpProvider{}
	err := p.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestNoOpProvider_ImplementsInterface(_ *testing.T) {
	var _ Provider = (*NoOpProvider)(nil)
}
