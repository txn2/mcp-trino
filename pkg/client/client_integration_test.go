//go:build integration

package client

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

// Integration tests require a running Trino instance.
// Run with: go test -tags=integration -v ./pkg/client/...
//
// Start Trino with: make docker-trino
// Environment variables:
//   TRINO_HOST (default: localhost)
//   TRINO_PORT (default: 8080)
//   TRINO_USER (default: test)

func getTestConfig() Config {
	host := os.Getenv("TRINO_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 8080
	if p := os.Getenv("TRINO_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	user := os.Getenv("TRINO_USER")
	if user == "" {
		user = "test"
	}

	return Config{
		Host:    host,
		Port:    port,
		User:    user,
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
		Timeout: 30 * time.Second,
		Source:  "integration-test",
	}
}

func setupIntegrationClient(t *testing.T) *Client {
	t.Helper()

	cfg := getTestConfig()
	client, err := New(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to Trino at %s:%d: %v", cfg.Host, cfg.Port, err)
	}

	// Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.ListCatalogs(ctx)
	if err != nil {
		client.Close()
		t.Skipf("Skipping integration test: Trino not ready: %v", err)
	}

	return client
}

func TestIntegration_ListCatalogs(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	catalogs, err := client.ListCatalogs(ctx)
	if err != nil {
		t.Fatalf("ListCatalogs failed: %v", err)
	}

	if len(catalogs) == 0 {
		t.Error("Expected at least one catalog")
	}

	// memory catalog should always exist
	found := false
	for _, c := range catalogs {
		if c == "memory" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'memory' catalog to exist")
	}

	t.Logf("Found catalogs: %v", catalogs)
}

func TestIntegration_ListSchemas(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	schemas, err := client.ListSchemas(ctx, "memory")
	if err != nil {
		t.Fatalf("ListSchemas failed: %v", err)
	}

	if len(schemas) == 0 {
		t.Error("Expected at least one schema")
	}

	// default and information_schema should exist
	foundDefault := false
	foundInfoSchema := false
	for _, s := range schemas {
		if s == "default" {
			foundDefault = true
		}
		if s == "information_schema" {
			foundInfoSchema = true
		}
	}

	if !foundDefault {
		t.Error("Expected 'default' schema to exist")
	}
	if !foundInfoSchema {
		t.Error("Expected 'information_schema' schema to exist")
	}

	t.Logf("Found schemas in memory catalog: %v", schemas)
}

func TestIntegration_Query_SimpleSelect(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	result, err := client.Query(ctx, "SELECT 1 AS num, 'hello' AS greeting", QueryOptions{
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(result.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(result.Columns))
	}

	if len(result.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(result.Rows))
	}

	if result.Columns[0].Name != "num" {
		t.Errorf("Expected first column name 'num', got %s", result.Columns[0].Name)
	}

	if result.Columns[1].Name != "greeting" {
		t.Errorf("Expected second column name 'greeting', got %s", result.Columns[1].Name)
	}

	t.Logf("Query result: %d columns, %d rows, took %v", len(result.Columns), len(result.Rows), result.Stats.Duration)
}

func TestIntegration_Query_WithLimit(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()

	// Generate 100 rows, but limit to 10
	result, err := client.Query(ctx, "SELECT * FROM (VALUES 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20) AS t(x)", QueryOptions{
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(result.Rows) > 10 {
		t.Errorf("Expected at most 10 rows due to limit, got %d", len(result.Rows))
	}

	t.Logf("Query with limit: got %d rows", len(result.Rows))
}

func TestIntegration_Query_SystemTables(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	result, err := client.Query(ctx, "SELECT node_id, http_uri, node_version FROM system.runtime.nodes", QueryOptions{
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(result.Rows) == 0 {
		t.Error("Expected at least one node in system.runtime.nodes")
	}

	t.Logf("Found %d Trino nodes", len(result.Rows))
	for i, row := range result.Rows {
		t.Logf("  Node %d: %v", i, row)
	}
}

func TestIntegration_Explain(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		explainType ExplainType
	}{
		{"Logical", ExplainLogical},
		{"Distributed", ExplainDistributed},
		{"IO", ExplainIO},
		{"Validate", ExplainValidate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Explain(ctx, "SELECT 1", tt.explainType)
			if err != nil {
				t.Fatalf("Explain %s failed: %v", tt.explainType, err)
			}

			if result.Plan == "" {
				t.Error("Expected non-empty plan")
			}

			if result.Type != tt.explainType {
				t.Errorf("Expected type %s, got %s", tt.explainType, result.Type)
			}

			t.Logf("Explain %s plan length: %d chars", tt.explainType, len(result.Plan))
		})
	}
}

func TestIntegration_CreateAndQueryTable(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create a table in memory catalog
	tableName := "memory.default.test_integration_table"

	// Drop table if exists (ignore errors)
	_, _ = client.Query(ctx, "DROP TABLE IF EXISTS "+tableName, QueryOptions{})

	// Create table
	_, err := client.Query(ctx, "CREATE TABLE "+tableName+" (id INT, name VARCHAR)", QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert data
	_, err = client.Query(ctx, "INSERT INTO "+tableName+" VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie')", QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Query data
	result, err := client.Query(ctx, "SELECT * FROM "+tableName+" ORDER BY id", QueryOptions{})
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}

	if len(result.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(result.Rows))
	}

	// List tables - should include our new table
	tables, err := client.ListTables(ctx, "memory", "default")
	if err != nil {
		t.Fatalf("Failed to list tables: %v", err)
	}

	found := false
	for _, tbl := range tables {
		if tbl.Name == "test_integration_table" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find test_integration_table in table list")
	}

	// Describe table
	tableInfo, err := client.DescribeTable(ctx, "memory", "default", "test_integration_table")
	if err != nil {
		t.Fatalf("Failed to describe table: %v", err)
	}

	if len(tableInfo.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(tableInfo.Columns))
	}

	// Cleanup
	_, err = client.Query(ctx, "DROP TABLE "+tableName, QueryOptions{})
	if err != nil {
		t.Logf("Warning: failed to drop test table: %v", err)
	}

	t.Log("Create, insert, query, describe, and drop table succeeded")
}

func TestIntegration_DescribeTable(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()

	// Describe a system table that always exists
	info, err := client.DescribeTable(ctx, "system", "runtime", "nodes")
	if err != nil {
		t.Fatalf("DescribeTable failed: %v", err)
	}

	if info.Catalog != "system" {
		t.Errorf("Expected catalog 'system', got %s", info.Catalog)
	}

	if info.Schema != "runtime" {
		t.Errorf("Expected schema 'runtime', got %s", info.Schema)
	}

	if info.Name != "nodes" {
		t.Errorf("Expected table name 'nodes', got %s", info.Name)
	}

	if len(info.Columns) == 0 {
		t.Error("Expected at least one column")
	}

	t.Logf("Table system.runtime.nodes has %d columns:", len(info.Columns))
	for _, col := range info.Columns {
		t.Logf("  - %s (%s)", col.Name, col.Type)
	}
}

func TestIntegration_QueryTimeout(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()

	// Use a very short timeout
	_, err := client.Query(ctx, "SELECT 1", QueryOptions{
		Timeout: 1 * time.Nanosecond,
	})

	// Should either succeed very fast or timeout
	// The important thing is it doesn't hang
	if err != nil {
		t.Logf("Query with tiny timeout returned error (expected): %v", err)
	} else {
		t.Log("Query with tiny timeout succeeded (also acceptable if very fast)")
	}
}
