package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
		errMsg  string
	}{
		{name: "empty string is valid", format: "", wantErr: false},
		{name: "json is valid", format: "json", wantErr: false},
		{name: "csv is valid", format: "csv", wantErr: false},
		{name: "markdown is valid", format: "markdown", wantErr: false},
		{name: "tsv is invalid", format: "tsv", wantErr: true, errMsg: `invalid format "tsv"`},
		{name: "table is invalid", format: "table", wantErr: true, errMsg: `invalid format "table"`},
		{name: "JSON uppercase is invalid", format: "JSON", wantErr: true, errMsg: `invalid format "JSON"`},
		{name: "Csv mixed case is invalid", format: "Csv", wantErr: true, errMsg: `invalid format "Csv"`},
		{name: "ndjson is invalid", format: "ndjson", wantErr: true, errMsg: `invalid format "ndjson"`},
		{name: "error names valid values", format: "bad", wantErr: true, errMsg: "json, csv, markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFormat(tt.format)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for format %q, got nil", tt.format)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for format %q: %v", tt.format, err)
				}
			}
		})
	}
}

func TestValidateExplainType(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		wantErr bool
		errMsg  string
	}{
		{name: "empty string is valid", typ: "", wantErr: false},
		{name: "logical is valid", typ: "logical", wantErr: false},
		{name: "distributed is valid", typ: "distributed", wantErr: false},
		{name: "io is valid", typ: "io", wantErr: false},
		{name: "validate is valid", typ: "validate", wantErr: false},
		{name: "LOGICAL uppercase is invalid", typ: "LOGICAL", wantErr: true, errMsg: `invalid explain type "LOGICAL"`},
		{name: "unknown is invalid", typ: "unknown", wantErr: true, errMsg: "must be one of"},
		{name: "error names valid values", typ: "bad", wantErr: true, errMsg: "logical, distributed, io, validate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExplainType(tt.typ)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for type %q, got nil", tt.typ)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for type %q: %v", tt.typ, err)
				}
			}
		})
	}
}

func TestFormatOutput(t *testing.T) {
	qo := &QueryOutput{
		Columns:  []QueryColumn{{Name: "id", Type: "INTEGER"}, {Name: "name", Type: "VARCHAR"}},
		Rows:     []map[string]any{{"id": 1, "name": "Alice"}},
		RowCount: 1,
		Stats:    QueryStats{RowCount: 1, DurationMs: 50},
	}

	tests := []struct {
		name     string
		format   string
		contains string
	}{
		{name: "empty defaults to json", format: "", contains: "Alice"},
		{name: "json format", format: "json", contains: "Alice"},
		{name: "csv format", format: "csv", contains: "id,name"},
		{name: "markdown format", format: "markdown", contains: "| id | name |"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatOutput(qo, tt.format)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got:\n%s", tt.contains, output)
			}
		})
	}

	t.Run("unsupported format returns error", func(t *testing.T) {
		_, err := formatOutput(qo, "tsv")
		if err == nil {
			t.Error("expected error for unsupported format")
		}
	})
}

func TestIsStringColumnType(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"VARCHAR", true},
		{"varchar", true},
		{"varchar(65535)", true},
		{"VARCHAR(255)", true},
		{"CHAR", true},
		{"char", true},
		{"char(10)", true},
		{"CHAR(1)", true},
		{"JSON", true},
		{"json", true},
		{"BIGINT", false},
		{"INTEGER", false},
		{"BOOLEAN", false},
		{"TIMESTAMP", false},
		{"VARBINARY", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := isStringColumnType(tt.typeName)
			if got != tt.want {
				t.Errorf("isStringColumnType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestUnwrapJSONColumn(t *testing.T) {
	t.Run("unwraps JSON object and sets column type", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "result", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"result": `{"took":2,"count":42}`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1, DurationMs: 10},
		}

		unwrapJSONColumn(qo)

		if qo.Columns[0].Type != columnTypeJSON {
			t.Errorf("expected column type JSON, got %s", qo.Columns[0].Type)
		}
		m, ok := qo.Rows[0]["result"].(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", qo.Rows[0]["result"])
		}
		if m["took"] != float64(2) {
			t.Errorf("expected took=2, got %v", m["took"])
		}
	})

	t.Run("unwraps JSON array", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "data", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"data": `[1,2,3]`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}

		unwrapJSONColumn(qo)

		if qo.Columns[0].Type != columnTypeJSON {
			t.Errorf("expected column type JSON, got %s", qo.Columns[0].Type)
		}
		arr, ok := qo.Rows[0]["data"].([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", qo.Rows[0]["data"])
		}
		if len(arr) != 3 {
			t.Errorf("expected 3 elements, got %d", len(arr))
		}
	})

	t.Run("accepts varchar with length constraint", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "varchar(65535)"}},
			Rows:     []map[string]any{{"r": `{"k":"v"}`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != columnTypeJSON {
			t.Errorf("expected JSON, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("accepts CHAR type", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "CHAR(1000)"}},
			Rows:     []map[string]any{{"r": `{"k":"v"}`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != columnTypeJSON {
			t.Errorf("expected JSON, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("accepts JSON type", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "JSON"}},
			Rows:     []map[string]any{{"r": `{"k":"v"}`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != columnTypeJSON {
			t.Errorf("expected JSON, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for scalar JSON string", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"r": `"hello"`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "VARCHAR" {
			t.Errorf("expected VARCHAR unchanged, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for scalar JSON number", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"r": `42`}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "VARCHAR" {
			t.Errorf("expected VARCHAR unchanged, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for multiple columns", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "id", Type: "INTEGER"}, {Name: "name", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"id": 1, "name": "Alice"}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "INTEGER" {
			t.Errorf("expected INTEGER unchanged, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for multiple rows", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"r": `{"a":1}`}, {"r": `{"b":2}`}},
			RowCount: 2,
			Stats:    QueryStats{RowCount: 2},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "VARCHAR" {
			t.Errorf("expected VARCHAR unchanged, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for non-string type", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "count", Type: "BIGINT"}},
			Rows:     []map[string]any{{"count": 42}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "BIGINT" {
			t.Errorf("expected BIGINT unchanged, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for non-JSON string", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "VARCHAR"}},
			Rows:     []map[string]any{{"r": "hello world"}},
			RowCount: 1,
			Stats:    QueryStats{RowCount: 1},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "VARCHAR" {
			t.Errorf("expected VARCHAR unchanged, got %s", qo.Columns[0].Type)
		}
		if qo.Rows[0]["r"] != "hello world" {
			t.Errorf("expected row value unchanged")
		}
	})

	t.Run("no-op for zero rows", func(t *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{{Name: "r", Type: "VARCHAR"}},
			Rows:     []map[string]any{},
			RowCount: 0,
			Stats:    QueryStats{RowCount: 0},
		}
		unwrapJSONColumn(qo)
		if qo.Columns[0].Type != "VARCHAR" {
			t.Errorf("expected VARCHAR unchanged, got %s", qo.Columns[0].Type)
		}
	})

	t.Run("no-op for empty columns", func(_ *testing.T) {
		qo := &QueryOutput{
			Columns:  []QueryColumn{},
			Rows:     []map[string]any{},
			RowCount: 0,
			Stats:    QueryStats{},
		}
		unwrapJSONColumn(qo) // should not panic
	})
}

func TestStringifyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"nil", nil, "<nil>"},
		{"map", map[string]any{"k": "v"}, `{"k":"v"}`},
		{"slice", []any{1, 2, 3}, `[1,2,3]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringifyValue(tt.input)
			if got != tt.expected {
				t.Errorf("stringifyValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUnwrapJSONColumn_JSONFormatOutput(t *testing.T) {
	// Verify that after unwrap, JSON output contains the parsed object inline
	// and the column type is "JSON" — same envelope, no double-encoding.
	qo := &QueryOutput{
		Columns:  []QueryColumn{{Name: "result", Type: "VARCHAR"}},
		Rows:     []map[string]any{{"result": `{"took":2,"aggs":{"count":42}}`}},
		RowCount: 1,
		Stats:    QueryStats{RowCount: 1, DurationMs: 10},
	}

	unwrapJSONColumn(qo)

	output, err := formatOutput(qo, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The JSON output should contain the parsed object inline (no escaping)
	if strings.Contains(output, `\"took\"`) {
		t.Error("JSON output still contains escaped quotes — value was not unwrapped")
	}
	if !strings.Contains(output, `"took"`) {
		t.Error("JSON output should contain the unwrapped 'took' field")
	}
	if !strings.Contains(output, `"type": "JSON"`) {
		t.Error("JSON output should show column type as JSON")
	}

	// Verify round-trip: parse the output and check structure
	var parsed QueryOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if parsed.Columns[0].Type != "JSON" {
		t.Errorf("expected column type JSON, got %s", parsed.Columns[0].Type)
	}
	if parsed.Stats.DurationMs != 10 {
		t.Errorf("expected duration_ms=10, got %d", parsed.Stats.DurationMs)
	}
}

func TestUnwrapJSONColumn_CSVFormatOutput(t *testing.T) {
	// After unwrap, CSV should contain the compact JSON string
	qo := &QueryOutput{
		Columns:  []QueryColumn{{Name: "result", Type: "VARCHAR"}},
		Rows:     []map[string]any{{"result": `{"key":"value"}`}},
		RowCount: 1,
		Stats:    QueryStats{RowCount: 1, DurationMs: 5},
	}

	unwrapJSONColumn(qo)

	output, err := formatOutput(qo, "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CSV should contain the compact JSON as a quoted value
	if !strings.Contains(output, `"key"`) {
		t.Errorf("CSV output should contain JSON key, got:\n%s", output)
	}
}
