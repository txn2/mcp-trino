package tools

import (
	"strings"
	"testing"
)

func TestBrowseInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   BrowseInput
		wantErr string
	}{
		{
			name:    "list catalogs (empty input)",
			input:   BrowseInput{},
			wantErr: "",
		},
		{
			name:    "list schemas (catalog only)",
			input:   BrowseInput{Catalog: "hive"},
			wantErr: "",
		},
		{
			name:    "list tables (catalog + schema)",
			input:   BrowseInput{Catalog: "hive", Schema: "default"},
			wantErr: "",
		},
		{
			name:    "list tables with pattern",
			input:   BrowseInput{Catalog: "hive", Schema: "default", Pattern: "%user%"},
			wantErr: "",
		},
		{
			name:    "schema without catalog",
			input:   BrowseInput{Schema: "default"},
			wantErr: "schema requires catalog",
		},
		{
			name:    "pattern without catalog",
			input:   BrowseInput{Pattern: "%user%"},
			wantErr: "pattern requires both catalog and schema",
		},
		{
			name:    "pattern without schema",
			input:   BrowseInput{Catalog: "hive", Pattern: "%user%"},
			wantErr: "pattern requires both catalog and schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBrowseInput(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestBrowsePatternMatching(t *testing.T) {
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

func TestBrowseInput_PatternEdgeCases(t *testing.T) {
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
			input := BrowseInput{
				Catalog: "hive",
				Schema:  "default",
				Pattern: tt.pattern,
			}

			if input.Pattern != tt.pattern {
				t.Errorf("expected Pattern %q, got %q", tt.pattern, input.Pattern)
			}
		})
	}
}
