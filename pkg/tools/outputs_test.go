package tools

import (
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
)

func TestBuildQueryOutput(t *testing.T) {
	result := &client.QueryResult{
		Columns: []client.ColumnInfo{
			{Name: "id", Type: "bigint"},
			{Name: "name", Type: "varchar"},
		},
		Rows: []map[string]any{
			{"id": 1, "name": "alice"},
			{"id": 2, "name": "bob"},
		},
		Stats: client.QueryStats{
			RowCount:     2,
			DurationMs:   42,
			Truncated:    false,
			LimitApplied: 1000,
		},
	}

	out := buildQueryOutput(result)

	if len(out.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(out.Columns))
	}
	if out.Columns[0].Name != "id" || out.Columns[0].Type != "bigint" {
		t.Errorf("unexpected column 0: %+v", out.Columns[0])
	}
	if out.Columns[1].Name != "name" || out.Columns[1].Type != "varchar" {
		t.Errorf("unexpected column 1: %+v", out.Columns[1])
	}
	if len(out.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(out.Rows))
	}
	if out.RowCount != 2 {
		t.Errorf("expected RowCount=2, got %d", out.RowCount)
	}
	if out.Stats.DurationMs != 42 {
		t.Errorf("expected DurationMs=42, got %d", out.Stats.DurationMs)
	}
	if out.Stats.Truncated {
		t.Error("expected Truncated=false")
	}
	if out.Stats.LimitApplied != 1000 {
		t.Errorf("expected LimitApplied=1000, got %d", out.Stats.LimitApplied)
	}
}

func TestBuildQueryOutput_Empty(t *testing.T) {
	result := &client.QueryResult{
		Stats: client.QueryStats{},
	}

	out := buildQueryOutput(result)

	if len(out.Columns) != 0 {
		t.Errorf("expected 0 columns, got %d", len(out.Columns))
	}
	if len(out.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(out.Rows))
	}
}

func TestBuildDescribeOutput(t *testing.T) {
	input := DescribeTableInput{
		Catalog: "hive",
		Schema:  "default",
		Table:   "users",
	}
	info := &client.TableInfo{
		Catalog: "hive",
		Schema:  "default",
		Name:    "users",
		Columns: []client.ColumnDef{
			{Name: "id", Type: "bigint", Nullable: "YES"},
			{Name: "email", Type: "varchar", Nullable: "NO", Comment: "user email"},
		},
	}

	out := buildDescribeOutput(input, info)

	if out.Catalog != "hive" || out.Schema != "default" || out.Table != "users" {
		t.Errorf("unexpected table identity: %s.%s.%s", out.Catalog, out.Schema, out.Table)
	}
	if out.Count != 2 {
		t.Errorf("expected Count=2, got %d", out.Count)
	}
	if len(out.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(out.Columns))
	}
	if out.Columns[0].Name != "id" || out.Columns[0].Type != "bigint" || out.Columns[0].Nullable != "YES" {
		t.Errorf("unexpected column 0: %+v", out.Columns[0])
	}
	if out.Columns[1].Comment != "user email" {
		t.Errorf("expected comment 'user email', got %q", out.Columns[1].Comment)
	}
}
