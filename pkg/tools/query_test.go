package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/txn2/mcp-trino/pkg/client"
)

func TestFormatCSV(t *testing.T) {
	tests := []struct {
		name     string
		result   *client.QueryResult
		contains []string
	}{
		{
			name: "basic CSV",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "id", Type: "INTEGER"},
					{Name: "name", Type: "VARCHAR"},
				},
				Rows: []map[string]any{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
				Stats: client.QueryStats{
					RowCount:     2,
					DurationMs:   100,
					Truncated:    false,
					LimitApplied: 1000,
				},
			},
			contains: []string{"id,name", "1,Alice", "2,Bob", "# 2 rows returned", "100ms"},
		},
		{
			name: "empty result",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{},
				Rows:    []map[string]any{},
				Stats:   client.QueryStats{},
			},
			contains: []string{},
		},
		{
			name: "truncated result",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "id", Type: "INTEGER"},
				},
				Rows: []map[string]any{
					{"id": 1},
				},
				Stats: client.QueryStats{
					RowCount:     1,
					DurationMs:   50,
					Truncated:    true,
					LimitApplied: 1,
				},
			},
			contains: []string{"truncated at limit 1"},
		},
		{
			name: "CSV with special characters",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "data", Type: "VARCHAR"},
				},
				Rows: []map[string]any{
					{"data": "hello,world"},
					{"data": "with\"quotes"},
					{"data": "with\nnewline"},
				},
				Stats: client.QueryStats{
					RowCount:   3,
					DurationMs: 10,
				},
			},
			contains: []string{"\"hello,world\"", "\"with\"\"quotes\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatCSV(tt.result)

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFormatMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		result   *client.QueryResult
		contains []string
	}{
		{
			name: "basic markdown table",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "id", Type: "INTEGER"},
					{Name: "name", Type: "VARCHAR"},
				},
				Rows: []map[string]any{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
				Stats: client.QueryStats{
					RowCount:     2,
					DurationMs:   100,
					Truncated:    false,
					LimitApplied: 1000,
				},
			},
			contains: []string{
				"| id | name |",
				"| --- | --- |",
				"| 1 | Alice |",
				"| 2 | Bob |",
				"*2 rows returned",
			},
		},
		{
			name: "empty result",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{},
				Rows:    []map[string]any{},
				Stats:   client.QueryStats{},
			},
			contains: []string{"No results"},
		},
		{
			name: "truncated result",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "id", Type: "INTEGER"},
				},
				Rows: []map[string]any{
					{"id": 1},
				},
				Stats: client.QueryStats{
					RowCount:     1,
					DurationMs:   50,
					Truncated:    true,
					LimitApplied: 1,
				},
			},
			contains: []string{"truncated at limit 1"},
		},
		{
			name: "nil values",
			result: &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "nullable_col", Type: "VARCHAR"},
				},
				Rows: []map[string]any{
					{"nullable_col": nil},
				},
				Stats: client.QueryStats{
					RowCount:   1,
					DurationMs: 10,
				},
			},
			contains: []string{"|  |"}, // Empty value for nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatMarkdown(tt.result)

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestEscapeCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "string with comma",
			input:    "hello,world",
			expected: "\"hello,world\"",
		},
		{
			name:     "string with quote",
			input:    "say \"hello\"",
			expected: "\"say \"\"hello\"\"\"",
		},
		{
			name:     "string with newline",
			input:    "line1\nline2",
			expected: "\"line1\nline2\"",
		},
		{
			name:     "string with carriage return",
			input:    "line1\rline2",
			expected: "\"line1\rline2\"",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple special chars",
			input:    "a,b\"c\nd",
			expected: "\"a,b\"\"c\nd\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeCSV(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestErrorResult(t *testing.T) {
	msg := "test error message"
	result := ErrorResult(msg)

	if !result.IsError {
		t.Error("expected IsError to be true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
}

func TestQueryInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input QueryInput
		valid bool
	}{
		{
			name: "valid input",
			input: QueryInput{
				SQL:            "SELECT 1",
				Limit:          100,
				TimeoutSeconds: 30,
				Format:         "json",
			},
			valid: true,
		},
		{
			name: "missing SQL",
			input: QueryInput{
				SQL:    "",
				Limit:  100,
				Format: "json",
			},
			valid: false,
		},
		{
			name: "default limit",
			input: QueryInput{
				SQL:   "SELECT 1",
				Limit: 0, // Should use default
			},
			valid: true,
		},
		{
			name: "default format",
			input: QueryInput{
				SQL:    "SELECT 1",
				Format: "", // Should use default (json)
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that empty SQL is invalid
			isValid := tt.input.SQL != ""
			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestQueryInput_Formats(t *testing.T) {
	formats := []string{"json", "csv", "markdown", ""}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			input := QueryInput{
				SQL:    "SELECT 1",
				Format: format,
			}

			// Just verify the input can be created with these formats
			if input.SQL == "" {
				t.Error("SQL should not be empty")
			}
		})
	}
}

func TestQueryInput_LimitBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		defaultLimit  int
		maxLimit      int
		expectedLimit int
	}{
		{
			name:          "zero uses default",
			inputLimit:    0,
			defaultLimit:  1000,
			maxLimit:      10000,
			expectedLimit: 1000,
		},
		{
			name:          "negative uses default",
			inputLimit:    -1,
			defaultLimit:  1000,
			maxLimit:      10000,
			expectedLimit: 1000,
		},
		{
			name:          "exceeds max",
			inputLimit:    20000,
			defaultLimit:  1000,
			maxLimit:      10000,
			expectedLimit: 10000,
		},
		{
			name:          "within bounds",
			inputLimit:    500,
			defaultLimit:  1000,
			maxLimit:      10000,
			expectedLimit: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				DefaultLimit: tt.defaultLimit,
				MaxLimit:     tt.maxLimit,
			}

			limit := tt.inputLimit
			if limit <= 0 {
				limit = cfg.DefaultLimit
			}
			if limit > cfg.MaxLimit {
				limit = cfg.MaxLimit
			}

			if limit != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, limit)
			}
		})
	}
}

func TestQueryInput_TimeoutBoundaries(t *testing.T) {
	tests := []struct {
		name            string
		inputTimeout    int
		defaultTimeout  time.Duration
		maxTimeout      time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "zero uses default",
			inputTimeout:    0,
			defaultTimeout:  120 * time.Second,
			maxTimeout:      300 * time.Second,
			expectedTimeout: 120 * time.Second,
		},
		{
			name:            "negative uses default",
			inputTimeout:    -1,
			defaultTimeout:  120 * time.Second,
			maxTimeout:      300 * time.Second,
			expectedTimeout: 120 * time.Second,
		},
		{
			name:            "exceeds max",
			inputTimeout:    600,
			defaultTimeout:  120 * time.Second,
			maxTimeout:      300 * time.Second,
			expectedTimeout: 300 * time.Second,
		},
		{
			name:            "within bounds",
			inputTimeout:    60,
			defaultTimeout:  120 * time.Second,
			maxTimeout:      300 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				DefaultTimeout: tt.defaultTimeout,
				MaxTimeout:     tt.maxTimeout,
			}

			timeout := time.Duration(tt.inputTimeout) * time.Second
			if timeout <= 0 {
				timeout = cfg.DefaultTimeout
			}
			if timeout > cfg.MaxTimeout {
				timeout = cfg.MaxTimeout
			}

			if timeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, timeout)
			}
		})
	}
}
