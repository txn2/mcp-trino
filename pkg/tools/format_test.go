package tools

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
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

func TestFormatOutput(t *testing.T) {
	result := &client.QueryResult{
		Columns: []client.ColumnInfo{
			{Name: "id", Type: "INTEGER"},
			{Name: "name", Type: "VARCHAR"},
		},
		Rows: []map[string]any{
			{"id": 1, "name": "Alice"},
		},
		Stats: client.QueryStats{
			RowCount:   1,
			DurationMs: 50,
		},
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
			output, err := formatOutput(result, tt.format)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got:\n%s", tt.contains, output)
			}
		})
	}
}

func TestTryUnwrapJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   *client.QueryResult
		wantOK   bool
		wantJSON string // JSON representation of expected parsed value, empty if wantOK is false
	}{
		{
			name: "single VARCHAR column with valid JSON object",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows:    []map[string]any{{"result": `{"took":2,"aggregations":{"count":42}}`}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK:   true,
			wantJSON: `{"took":2,"aggregations":{"count":42}}`,
		},
		{
			name: "single VARCHAR column with valid JSON array",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "data", Type: "VARCHAR"}},
				Rows:    []map[string]any{{"data": `[1,2,3]`}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK:   true,
			wantJSON: `[1,2,3]`,
		},
		{
			name: "VARCHAR with length constraint",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "varchar(65535)"}},
				Rows:    []map[string]any{{"result": `{"key":"value"}`}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK:   true,
			wantJSON: `{"key":"value"}`,
		},
		{
			name: "multiple columns - no unwrap",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "id", Type: "INTEGER"},
					{Name: "name", Type: "VARCHAR"},
				},
				Rows:  []map[string]any{{"id": 1, "name": "Alice"}},
				Stats: client.QueryStats{RowCount: 1},
			},
			wantOK: false,
		},
		{
			name: "single column non-VARCHAR type - no unwrap",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "count", Type: "BIGINT"}},
				Rows:    []map[string]any{{"count": 42}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK: false,
		},
		{
			name: "multiple rows - no unwrap",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows: []map[string]any{
					{"result": `{"a":1}`},
					{"result": `{"b":2}`},
				},
				Stats: client.QueryStats{RowCount: 2},
			},
			wantOK: false,
		},
		{
			name: "zero rows - no unwrap",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows:    []map[string]any{},
				Stats:   client.QueryStats{RowCount: 0},
			},
			wantOK: false,
		},
		{
			name: "single VARCHAR column with non-JSON string",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows:    []map[string]any{{"result": "hello world"}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK: false,
		},
		{
			name: "single VARCHAR column with empty string",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows:    []map[string]any{{"result": ""}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK: false,
		},
		{
			name: "single VARCHAR column with nil value",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows:    []map[string]any{{"result": nil}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK: false,
		},
		{
			name: "single VARCHAR column with non-string value",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{{Name: "result", Type: "VARCHAR"}},
				Rows:    []map[string]any{{"result": 42}},
				Stats:   client.QueryStats{RowCount: 1},
			},
			wantOK: false,
		},
		{
			name: "no columns - no unwrap",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{},
				Rows:    []map[string]any{},
				Stats:   client.QueryStats{},
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, ok := tryUnwrapJSON(tt.result)
			if ok != tt.wantOK {
				t.Errorf("tryUnwrapJSON() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if tt.wantOK && tt.wantJSON != "" {
				got, err := json.Marshal(parsed)
				if err != nil {
					t.Fatalf("failed to marshal parsed result: %v", err)
				}
				// Compare as JSON to avoid whitespace differences
				var expected, actual any
				if err := json.Unmarshal([]byte(tt.wantJSON), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected JSON: %v", err)
				}
				if err := json.Unmarshal(got, &actual); err != nil {
					t.Fatalf("failed to unmarshal actual JSON: %v", err)
				}
				expectedBytes, errE := json.Marshal(expected)
				if errE != nil {
					t.Fatalf("failed to marshal expected: %v", errE)
				}
				actualBytes, errA := json.Marshal(actual)
				if errA != nil {
					t.Fatalf("failed to marshal actual: %v", errA)
				}
				if !bytes.Equal(expectedBytes, actualBytes) {
					t.Errorf("parsed JSON = %s, want %s", string(actualBytes), string(expectedBytes))
				}
			}
			if !tt.wantOK && parsed != nil {
				t.Errorf("expected nil parsed result when ok is false, got %v", parsed)
			}
		})
	}
}
