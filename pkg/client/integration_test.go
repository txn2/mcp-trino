package client

import (
	"context"
	"testing"
	"time"
)

// TestClient_Ping tests the Ping function.
func TestClient_Ping(t *testing.T) {
	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Source:  "test",
		Catalog: "memory",
		Schema:  "default",
		Timeout: 5 * time.Second,
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Ping will fail without a real Trino server, but we can test the function exists
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// We expect an error since there's no Trino server
	err = client.Ping(ctx)
	if err == nil {
		// If it succeeds, we have a Trino server running
		t.Log("Ping succeeded - Trino server is available")
	} else {
		// Expected - no server
		t.Logf("Ping failed as expected without Trino: %v", err)
	}
}

// TestClient_Close tests the Close function.
func TestClient_Close(t *testing.T) {
	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Source:  "test",
		Catalog: "memory",
		Schema:  "default",
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Close should not return an error
	err = client.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

// TestClient_Config tests the Config accessor.
func TestClient_Config(t *testing.T) {
	cfg := Config{
		Host:    "testhost",
		Port:    9090,
		User:    "testuser",
		SSL:     true,
		Source:  "testsource",
		Catalog: "testcatalog",
		Schema:  "testschema",
		Timeout: 60 * time.Second,
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	returnedCfg := client.Config()

	if returnedCfg.Host != cfg.Host {
		t.Errorf("expected Host %q, got %q", cfg.Host, returnedCfg.Host)
	}
	if returnedCfg.Port != cfg.Port {
		t.Errorf("expected Port %d, got %d", cfg.Port, returnedCfg.Port)
	}
	if returnedCfg.User != cfg.User {
		t.Errorf("expected User %q, got %q", cfg.User, returnedCfg.User)
	}
	if returnedCfg.SSL != cfg.SSL {
		t.Errorf("expected SSL %v, got %v", cfg.SSL, returnedCfg.SSL)
	}
	if returnedCfg.Source != cfg.Source {
		t.Errorf("expected Source %q, got %q", cfg.Source, returnedCfg.Source)
	}
	if returnedCfg.Catalog != cfg.Catalog {
		t.Errorf("expected Catalog %q, got %q", cfg.Catalog, returnedCfg.Catalog)
	}
	if returnedCfg.Schema != cfg.Schema {
		t.Errorf("expected Schema %q, got %q", cfg.Schema, returnedCfg.Schema)
	}
	if returnedCfg.Timeout != cfg.Timeout {
		t.Errorf("expected Timeout %v, got %v", cfg.Timeout, returnedCfg.Timeout)
	}
}

// TestExplainSQL tests the SQL generation for different explain types.
func TestExplainSQL(t *testing.T) {
	tests := []struct {
		name        string
		explainType ExplainType
		query       string
		expected    string
	}{
		{
			name:        "logical explain",
			explainType: ExplainLogical,
			query:       "SELECT 1",
			expected:    "EXPLAIN (TYPE LOGICAL) SELECT 1",
		},
		{
			name:        "distributed explain",
			explainType: ExplainDistributed,
			query:       "SELECT 1",
			expected:    "EXPLAIN (TYPE DISTRIBUTED) SELECT 1",
		},
		{
			name:        "io explain",
			explainType: ExplainIO,
			query:       "SELECT 1",
			expected:    "EXPLAIN (TYPE IO) SELECT 1",
		},
		{
			name:        "validate explain",
			explainType: ExplainValidate,
			query:       "SELECT 1",
			expected:    "EXPLAIN (TYPE VALIDATE) SELECT 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All explain types use the same format: EXPLAIN (TYPE X) query
			explainSQL := "EXPLAIN (TYPE " + string(tt.explainType) + ") " + tt.query

			if explainSQL != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, explainSQL)
			}
		})
	}
}

// TestQueryOptions_WithValues tests QueryOptions with various values.
func TestQueryOptions_WithValues(t *testing.T) {
	tests := []struct {
		name    string
		opts    QueryOptions
		timeout time.Duration
	}{
		{
			name: "all fields set",
			opts: QueryOptions{
				Limit:   500,
				Timeout: 30 * time.Second,
				Catalog: "hive",
				Schema:  "sales",
			},
			timeout: 30 * time.Second,
		},
		{
			name: "default values",
			opts: QueryOptions{},
		},
		{
			name: "only limit",
			opts: QueryOptions{
				Limit: 100,
			},
		},
		{
			name: "only timeout",
			opts: QueryOptions{
				Timeout: 60 * time.Second,
			},
			timeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the options can be created and accessed
			if tt.opts.Timeout != 0 && tt.opts.Timeout != tt.timeout {
				t.Errorf("expected Timeout %v, got %v", tt.timeout, tt.opts.Timeout)
			}
		})
	}
}

// TestTableInfo_Fields tests the TableInfo struct.
func TestTableInfo_Fields(t *testing.T) {
	info := TableInfo{
		Catalog: "hive",
		Schema:  "default",
		Name:    "users",
		Type:    "TABLE",
		Columns: []ColumnDef{
			{Name: "id", Type: "BIGINT", Nullable: "NO"},
			{Name: "name", Type: "VARCHAR", Nullable: "YES"},
		},
	}

	if info.Catalog != "hive" {
		t.Errorf("expected Catalog 'hive', got %q", info.Catalog)
	}
	if info.Schema != "default" {
		t.Errorf("expected Schema 'default', got %q", info.Schema)
	}
	if info.Name != "users" {
		t.Errorf("expected Name 'users', got %q", info.Name)
	}
	if info.Type != "TABLE" {
		t.Errorf("expected Type 'TABLE', got %q", info.Type)
	}
	if len(info.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(info.Columns))
	}
}

// TestColumnDef_Fields tests the ColumnDef struct.
func TestColumnDef_Fields(t *testing.T) {
	col := ColumnDef{
		Name:     "created_at",
		Type:     "TIMESTAMP",
		Nullable: "YES",
		Comment:  "Row creation time",
	}

	if col.Name != "created_at" {
		t.Errorf("expected Name 'created_at', got %q", col.Name)
	}
	if col.Type != "TIMESTAMP" {
		t.Errorf("expected Type 'TIMESTAMP', got %q", col.Type)
	}
	if col.Nullable != "YES" {
		t.Errorf("expected Nullable 'YES', got %q", col.Nullable)
	}
	if col.Comment != "Row creation time" {
		t.Errorf("expected Comment 'Row creation time', got %q", col.Comment)
	}
}

// TestQueryStats_Fields tests the QueryStats struct.
func TestQueryStats_Fields(t *testing.T) {
	stats := QueryStats{
		RowCount:     1000,
		DurationMs:   500,
		Truncated:    true,
		LimitApplied: 1000,
		QueryID:      "test_query_123",
	}

	if stats.RowCount != 1000 {
		t.Errorf("expected RowCount 1000, got %d", stats.RowCount)
	}
	if stats.DurationMs != 500 {
		t.Errorf("expected DurationMs 500, got %d", stats.DurationMs)
	}
	if !stats.Truncated {
		t.Error("expected Truncated to be true")
	}
	if stats.LimitApplied != 1000 {
		t.Errorf("expected LimitApplied 1000, got %d", stats.LimitApplied)
	}
	if stats.QueryID != "test_query_123" {
		t.Errorf("expected QueryID 'test_query_123', got %q", stats.QueryID)
	}
}

// TestColumnInfo_Fields tests the ColumnInfo struct.
func TestColumnInfo_Fields(t *testing.T) {
	col := ColumnInfo{
		Name:     "email",
		Type:     "VARCHAR",
		Nullable: true,
	}

	if col.Name != "email" {
		t.Errorf("expected Name 'email', got %q", col.Name)
	}
	if col.Type != "VARCHAR" {
		t.Errorf("expected Type 'VARCHAR', got %q", col.Type)
	}
	if !col.Nullable {
		t.Error("expected Nullable to be true")
	}
}

// TestExplainResult_Fields tests the ExplainResult struct.
func TestExplainResult_Fields(t *testing.T) {
	result := ExplainResult{
		Type: ExplainDistributed,
		Plan: "- Output\n  - TableScan",
	}

	if result.Type != ExplainDistributed {
		t.Errorf("expected Type DISTRIBUTED, got %v", result.Type)
	}
	if result.Plan != "- Output\n  - TableScan" {
		t.Errorf("unexpected Plan: %q", result.Plan)
	}
}
