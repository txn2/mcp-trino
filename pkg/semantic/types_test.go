package semantic

import (
	"testing"
	"time"
)

func TestTableIdentifier_Key(t *testing.T) {
	tests := []struct {
		name     string
		table    TableIdentifier
		expected string
	}{
		{
			name: "full identifier",
			table: TableIdentifier{
				Connection: "prod",
				Catalog:    "hive",
				Schema:     "analytics",
				Table:      "users",
			},
			expected: "prod:hive.analytics.users",
		},
		{
			name: "no connection",
			table: TableIdentifier{
				Catalog: "hive",
				Schema:  "analytics",
				Table:   "users",
			},
			expected: "hive.analytics.users",
		},
		{
			name: "minimal",
			table: TableIdentifier{
				Catalog: "memory",
				Schema:  "default",
				Table:   "test",
			},
			expected: "memory.default.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.table.Key()
			if got != tt.expected {
				t.Errorf("Key() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTableIdentifier_String(t *testing.T) {
	table := TableIdentifier{
		Catalog: "hive",
		Schema:  "analytics",
		Table:   "users",
	}
	expected := "hive.analytics.users"
	got := table.String()
	if got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestColumnIdentifier_String(t *testing.T) {
	tests := []struct {
		name     string
		column   ColumnIdentifier
		expected string
	}{
		{
			name: "with connection",
			column: ColumnIdentifier{
				TableIdentifier: TableIdentifier{
					Connection: "prod",
					Catalog:    "hive",
					Schema:     "analytics",
					Table:      "users",
				},
				Column: "user_id",
			},
			expected: "prod:hive.analytics.users.user_id",
		},
		{
			name: "without connection",
			column: ColumnIdentifier{
				TableIdentifier: TableIdentifier{
					Catalog: "hive",
					Schema:  "analytics",
					Table:   "users",
				},
				Column: "email",
			},
			expected: "hive.analytics.users.email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.column.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLineageDirection_Values(t *testing.T) {
	tests := []struct {
		dir      LineageDirection
		expected string
	}{
		{LineageUpstream, "upstream"},
		{LineageDownstream, "downstream"},
		{LineageDirection("other"), "other"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dir), func(t *testing.T) {
			got := string(tt.dir)
			if got != tt.expected {
				t.Errorf("LineageDirection = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSearchFilter_Defaults(t *testing.T) {
	filter := SearchFilter{}

	if filter.Limit != 0 {
		t.Errorf("default Limit should be 0, got %d", filter.Limit)
	}
	if filter.IncludeDeprecated {
		t.Error("default IncludeDeprecated should be false")
	}
}

func TestTableContext_Fields(t *testing.T) {
	now := time.Now()
	score := 95.5

	tc := &TableContext{
		Identifier: TableIdentifier{
			Catalog: "hive",
			Schema:  "analytics",
			Table:   "users",
		},
		Description: "User accounts table",
		Ownership: &Ownership{
			Owners: []Owner{
				{ID: "user1", Name: "Alice", Type: "user", Role: "owner"},
			},
		},
		Tags: []Tag{
			{Name: "pii", Description: "Contains PII data"},
		},
		GlossaryTerms: []GlossaryTerm{
			{URN: "urn:term:1", Name: "Customer"},
		},
		Domain: &Domain{
			URN:  "urn:domain:1",
			Name: "Customer Analytics",
		},
		Deprecation: &Deprecation{
			Deprecated: true,
			Note:       "Use v2 table",
			ReplacedBy: "users_v2",
		},
		Quality: &DataQuality{
			Score: &score,
		},
		CustomProperties: map[string]any{
			"custom_key": "custom_value",
		},
		Source:    "test",
		FetchedAt: now,
	}

	if tc.Identifier.Table != "users" {
		t.Errorf("unexpected table name: %s", tc.Identifier.Table)
	}
	if tc.Description != "User accounts table" {
		t.Errorf("unexpected description: %s", tc.Description)
	}
	if len(tc.Ownership.Owners) != 1 {
		t.Errorf("unexpected owner count: %d", len(tc.Ownership.Owners))
	}
	if tc.Ownership.Owners[0].Name != "Alice" {
		t.Errorf("unexpected owner name: %s", tc.Ownership.Owners[0].Name)
	}
	if len(tc.Tags) != 1 {
		t.Errorf("unexpected tag count: %d", len(tc.Tags))
	}
	if tc.Domain.Name != "Customer Analytics" {
		t.Errorf("unexpected domain name: %s", tc.Domain.Name)
	}
	if !tc.Deprecation.Deprecated {
		t.Error("expected deprecated to be true")
	}
	if *tc.Quality.Score != 95.5 {
		t.Errorf("unexpected quality score: %f", *tc.Quality.Score)
	}
	if tc.Source != "test" {
		t.Errorf("unexpected source: %s", tc.Source)
	}
}

func TestColumnContext_Fields(t *testing.T) {
	cc := &ColumnContext{
		Identifier: ColumnIdentifier{
			TableIdentifier: TableIdentifier{
				Catalog: "hive",
				Schema:  "analytics",
				Table:   "users",
			},
			Column: "email",
		},
		Description:      "User email address",
		Tags:             []Tag{{Name: "pii"}},
		GlossaryTerms:    []GlossaryTerm{{Name: "Email"}},
		IsSensitive:      true,
		SensitivityLevel: "high",
		Source:           "test",
	}

	if cc.Identifier.Column != "email" {
		t.Errorf("unexpected column name: %s", cc.Identifier.Column)
	}
	if !cc.IsSensitive {
		t.Error("expected IsSensitive to be true")
	}
	if cc.SensitivityLevel != "high" {
		t.Errorf("unexpected sensitivity level: %s", cc.SensitivityLevel)
	}
}

func TestLineageInfo_Fields(t *testing.T) {
	li := &LineageInfo{
		Table: TableIdentifier{
			Catalog: "hive",
			Schema:  "analytics",
			Table:   "users",
		},
		Direction: LineageUpstream,
		Edges: []LineageEdge{
			{
				SourceTable: TableIdentifier{Catalog: "hive", Schema: "raw", Table: "users_raw"},
				TargetTable: TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
			},
		},
		Source: "datahub",
	}

	if li.Direction != LineageUpstream {
		t.Errorf("unexpected direction: %s", li.Direction)
	}
	if len(li.Edges) != 1 {
		t.Errorf("unexpected edge count: %d", len(li.Edges))
	}
	if li.Edges[0].SourceTable.Table != "users_raw" {
		t.Errorf("unexpected source table: %s", li.Edges[0].SourceTable.Table)
	}
}

func TestGlossaryTerm_Fields(t *testing.T) {
	gt := &GlossaryTerm{
		URN:          "urn:li:glossaryTerm:mrr",
		Name:         "MRR",
		Definition:   "Monthly Recurring Revenue",
		RelatedTerms: []string{"ARR", "Revenue"},
		Source:       "datahub",
	}

	if gt.URN != "urn:li:glossaryTerm:mrr" {
		t.Errorf("unexpected URN: %s", gt.URN)
	}
	if gt.Name != "MRR" {
		t.Errorf("unexpected name: %s", gt.Name)
	}
	if len(gt.RelatedTerms) != 2 {
		t.Errorf("unexpected related terms count: %d", len(gt.RelatedTerms))
	}
}

func TestFreshnessInfo_Fields(t *testing.T) {
	lastUpdated := time.Now().Add(-1 * time.Hour)
	fi := &FreshnessInfo{
		LastUpdated:       &lastUpdated,
		ExpectedFrequency: "daily",
		Status:            "fresh",
	}

	if fi.ExpectedFrequency != "daily" {
		t.Errorf("unexpected expected frequency: %s", fi.ExpectedFrequency)
	}
	if fi.Status != "fresh" {
		t.Errorf("unexpected status: %s", fi.Status)
	}
	if fi.LastUpdated == nil {
		t.Error("expected LastUpdated to be set")
	}
}
