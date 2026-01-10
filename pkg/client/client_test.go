package client

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestNew_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "missing host",
			config: Config{
				Host: "",
				Port: 8080,
				User: "admin",
			},
		},
		{
			name: "missing user",
			config: Config{
				Host: "localhost",
				Port: 8080,
				User: "",
			},
		},
		{
			name: "invalid port",
			config: Config{
				Host: "localhost",
				Port: 0,
				User: "admin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config)
			if err == nil {
				t.Error("expected error for invalid config")
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

func TestNew_ValidConfig(t *testing.T) {
	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "admin",
		SSL:     false,
		Source:  "test",
		Catalog: "memory",
		Schema:  "default",
		Timeout: 30 * time.Second,
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	// Verify config is stored
	storedCfg := client.Config()
	if storedCfg.Host != cfg.Host {
		t.Errorf("expected Host %q, got %q", cfg.Host, storedCfg.Host)
	}
	if storedCfg.Port != cfg.Port {
		t.Errorf("expected Port %d, got %d", cfg.Port, storedCfg.Port)
	}
}

func TestDefaultQueryOptions(t *testing.T) {
	opts := DefaultQueryOptions()

	if opts.Limit != 1000 {
		t.Errorf("expected default Limit 1000, got %d", opts.Limit)
	}
	if opts.Timeout != 0 {
		t.Errorf("expected default Timeout 0 (use client default), got %v", opts.Timeout)
	}
	if opts.Catalog != "" {
		t.Errorf("expected empty Catalog, got %q", opts.Catalog)
	}
	if opts.Schema != "" {
		t.Errorf("expected empty Schema, got %q", opts.Schema)
	}
}

func TestExplainType_Constants(t *testing.T) {
	tests := []struct {
		explainType ExplainType
		expected    string
	}{
		{ExplainLogical, "LOGICAL"},
		{ExplainDistributed, "DISTRIBUTED"},
		{ExplainIO, "IO"},
		{ExplainValidate, "VALIDATE"},
	}

	for _, tt := range tests {
		t.Run(string(tt.explainType), func(t *testing.T) {
			if string(tt.explainType) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.explainType))
			}
		})
	}
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int value",
			input:    42,
			expected: 42,
		},
		{
			name:     "float value",
			input:    3.14,
			expected: 3.14,
		},
		{
			name:     "bool value",
			input:    true,
			expected: true,
		},
		{
			name:     "bytes as string",
			input:    []byte("hello world"),
			expected: "hello world",
		},
		{
			name:     "bytes as JSON object",
			input:    []byte(`{"key": "value"}`),
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "bytes as JSON array",
			input:    []byte(`[1, 2, 3]`),
			expected: []any{float64(1), float64(2), float64(3)},
		},
		{
			name:     "time value",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: "2024-01-15T10:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertValue(tt.input)

			// For complex types, use JSON comparison
			expectedJSON, err := json.Marshal(tt.expected)
			if err != nil {
				t.Fatalf("failed to marshal expected value: %v", err)
			}
			resultJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal result: %v", err)
			}

			if !bytes.Equal(expectedJSON, resultJSON) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestQueryResult_JSON(t *testing.T) {
	result := &QueryResult{
		Columns: []ColumnInfo{
			{Name: "id", Type: "INTEGER", Nullable: false},
			{Name: "name", Type: "VARCHAR", Nullable: true},
		},
		Rows: []map[string]any{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		},
		Stats: QueryStats{
			RowCount:     2,
			Duration:     100 * time.Millisecond,
			Truncated:    false,
			LimitApplied: 1000,
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal QueryResult: %v", err)
	}

	var decoded QueryResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal QueryResult: %v", err)
	}

	if len(decoded.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(decoded.Columns))
	}
	if len(decoded.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(decoded.Rows))
	}
	if decoded.Stats.RowCount != 2 {
		t.Errorf("expected RowCount 2, got %d", decoded.Stats.RowCount)
	}
}

func TestColumnInfo_JSON(t *testing.T) {
	col := ColumnInfo{
		Name:     "user_id",
		Type:     "BIGINT",
		Nullable: false,
	}

	data, err := json.Marshal(col)
	if err != nil {
		t.Fatalf("failed to marshal ColumnInfo: %v", err)
	}

	expected := `{"name":"user_id","type":"BIGINT","nullable":false}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestExplainResult_JSON(t *testing.T) {
	result := &ExplainResult{
		Type: ExplainLogical,
		Plan: "- Output\n  - TableScan",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal ExplainResult: %v", err)
	}

	var decoded ExplainResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ExplainResult: %v", err)
	}

	if decoded.Type != ExplainLogical {
		t.Errorf("expected Type LOGICAL, got %s", decoded.Type)
	}
	if decoded.Plan != result.Plan {
		t.Errorf("expected Plan %q, got %q", result.Plan, decoded.Plan)
	}
}

func TestTableInfo_JSON(t *testing.T) {
	info := TableInfo{
		Catalog: "hive",
		Schema:  "default",
		Name:    "users",
		Type:    "TABLE",
		Columns: []ColumnDef{
			{Name: "id", Type: "BIGINT", Nullable: "NO", Comment: "Primary key"},
			{Name: "name", Type: "VARCHAR", Nullable: "YES", Comment: ""},
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal TableInfo: %v", err)
	}

	var decoded TableInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal TableInfo: %v", err)
	}

	if decoded.Catalog != "hive" {
		t.Errorf("expected Catalog 'hive', got %q", decoded.Catalog)
	}
	if len(decoded.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(decoded.Columns))
	}
}

func TestColumnDef_JSON(t *testing.T) {
	col := ColumnDef{
		Name:     "created_at",
		Type:     "TIMESTAMP",
		Nullable: "YES",
		Comment:  "Creation timestamp",
	}

	data, err := json.Marshal(col)
	if err != nil {
		t.Fatalf("failed to marshal ColumnDef: %v", err)
	}

	var decoded ColumnDef
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ColumnDef: %v", err)
	}

	if decoded.Name != "created_at" {
		t.Errorf("expected Name 'created_at', got %q", decoded.Name)
	}
	if decoded.Comment != "Creation timestamp" {
		t.Errorf("expected Comment 'Creation timestamp', got %q", decoded.Comment)
	}
}

func TestQueryStats_JSON(t *testing.T) {
	stats := QueryStats{
		RowCount:     100,
		Duration:     500 * time.Millisecond,
		Truncated:    true,
		LimitApplied: 100,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal QueryStats: %v", err)
	}

	var decoded QueryStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal QueryStats: %v", err)
	}

	if decoded.RowCount != 100 {
		t.Errorf("expected RowCount 100, got %d", decoded.RowCount)
	}
	if !decoded.Truncated {
		t.Error("expected Truncated to be true")
	}
}

func TestQueryOptions_Defaults(t *testing.T) {
	opts := QueryOptions{}

	// Default Limit should be 0 (caller should set default)
	if opts.Limit != 0 {
		t.Errorf("expected zero Limit for empty struct, got %d", opts.Limit)
	}

	// DefaultQueryOptions should provide defaults
	defaultOpts := DefaultQueryOptions()
	if defaultOpts.Limit != 1000 {
		t.Errorf("expected default Limit 1000, got %d", defaultOpts.Limit)
	}
}
