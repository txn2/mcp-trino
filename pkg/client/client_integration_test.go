//go:build integration

package client

import (
	"context"
	"errors"
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

	t.Logf("Query result: %d columns, %d rows, took %dms", len(result.Columns), len(result.Rows), result.Stats.DurationMs)
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

// TestIntegration_Query_ExactValues covers #94: a query's values and declared
// types reach the caller exactly, on the rendered path and with RawValues.
func TestIntegration_Query_ExactValues(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	const sql = `SELECT TIMESTAMP '2024-05-01 12:34:56.789123' AS ts,
		CAST(1.5 AS DECIMAL(12,2)) AS amount,
		X'0001FF' AS blob`
	// The driver places a zoneless TIMESTAMP in the process's local zone.
	wantTS := time.Date(2024, 5, 1, 12, 34, 56, 789123000, time.Local)

	rendered, err := client.Query(ctx, sql, DefaultQueryOptions())
	if err != nil {
		t.Fatalf("rendered query failed: %v", err)
	}
	raw, err := client.Query(ctx, sql, QueryOptions{Limit: 1, RawValues: true})
	if err != nil {
		t.Fatalf("raw query failed: %v", err)
	}

	for _, r := range []*QueryResult{rendered, raw} {
		if len(r.Columns) != 3 || len(r.Rows) != 1 {
			t.Fatalf("got %d columns and %d rows, want 3 and 1", len(r.Columns), len(r.Rows))
		}
		ts, amount := r.Columns[0], r.Columns[1]
		if ts.Precision != 6 {
			t.Errorf("timestamp precision: got %d, want 6", ts.Precision)
		}
		if amount.Precision != 12 || amount.Scale != 2 {
			t.Errorf("decimal: got (%d,%d), want (12,2)", amount.Precision, amount.Scale)
		}
	}

	if got := rendered.Rows[0]["ts"]; got != wantTS.Format(time.RFC3339Nano) {
		t.Errorf("rendered timestamp: got %#v", got)
	}
	if got := rendered.Rows[0]["blob"]; got != "\x00\x01\xff" {
		t.Errorf("rendered varbinary: got %#v, want the text form", got)
	}

	if got, ok := raw.Rows[0]["ts"].(time.Time); !ok || !got.Equal(wantTS) {
		t.Errorf("raw timestamp: got %#v, want %v", raw.Rows[0]["ts"], wantTS)
	}
	if got, ok := raw.Rows[0]["blob"].([]byte); !ok || string(got) != "\x00\x01\xff" {
		t.Errorf("raw varbinary: got %#v, want []byte{0x00, 0x01, 0xff}", raw.Rows[0]["blob"])
	}
	if got := raw.Rows[0]["amount"]; got != "1.50" {
		t.Errorf("raw decimal: got %#v, want \"1.50\"", got)
	}
}

// TestIntegration_QueryStream covers #69: QueryStream reads a result a row at
// a time, with no cap unless a limit is given, and stops when ctx is cancelled.
func TestIntegration_QueryStream(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	count, err := client.Query(ctx, "SELECT count(*) AS n FROM tpch.tiny.lineitem", DefaultQueryOptions())
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	want, ok := count.Rows[0]["n"].(int64)
	if !ok || want <= 1000 {
		t.Fatalf("expected more rows than Query's default limit, got %#v", count.Rows[0]["n"])
	}

	t.Run("no limit reads every row", func(t *testing.T) {
		cur, err := client.QueryStream(ctx, "SELECT orderkey, linenumber FROM tpch.tiny.lineitem", QueryOptions{})
		if err != nil {
			t.Fatalf("QueryStream failed: %v", err)
		}
		defer func() { _ = cur.Close() }()

		if len(cur.Columns()) != 2 {
			t.Fatalf("columns before Next: got %+v", cur.Columns())
		}
		var n int64
		for cur.Next() {
			n++
		}
		if err := cur.Err(); err != nil {
			t.Fatalf("iteration failed: %v", err)
		}
		if n != want || cur.Stats().Truncated {
			t.Errorf("got %d rows (truncated %v), want %d untruncated", n, cur.Stats().Truncated, want)
		}
	})

	t.Run("limit truncates", func(t *testing.T) {
		cur, err := client.QueryStream(ctx, "SELECT orderkey FROM tpch.tiny.lineitem", QueryOptions{Limit: 10})
		if err != nil {
			t.Fatalf("QueryStream failed: %v", err)
		}
		defer func() { _ = cur.Close() }()

		n := 0
		for cur.Next() {
			n++
		}
		if n != 10 || !cur.Stats().Truncated || cur.Err() != nil {
			t.Errorf("got %d rows, truncated %v, err %v; want 10, true, nil", n, cur.Stats().Truncated, cur.Err())
		}
	})

	t.Run("cancelling ctx ends the stream", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		defer cancel()
		cur, err := client.QueryStream(cctx, "SELECT orderkey FROM tpch.sf1.lineitem", QueryOptions{})
		if err != nil {
			t.Fatalf("QueryStream failed: %v", err)
		}
		defer func() { _ = cur.Close() }()

		if !cur.Next() {
			t.Fatalf("expected a first row, err %v", cur.Err())
		}
		cancel()

		n := 1
		for cur.Next() {
			n++
		}
		if !errors.Is(cur.Err(), context.Canceled) {
			t.Errorf("Err after cancel: got %v, want context.Canceled", cur.Err())
		}
		// tpch.sf1.lineitem holds about six million rows.
		if n >= 1_000_000 {
			t.Errorf("read %d rows after cancelling; the stream did not stop", n)
		}
		t.Logf("read %d rows before the cancelled stream ended", n)
	})
}

// TestIntegration_SessionCatalogSchema checks the session catalog and schema
// Trino reports for the configured values and for per-query overrides.
func TestIntegration_SessionCatalogSchema(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	tests := []struct {
		name                    string
		opts                    QueryOptions
		wantCatalog, wantSchema any
	}{
		{name: "configured", wantCatalog: "memory", wantSchema: "default"},
		{name: "catalog and schema override", opts: QueryOptions{Catalog: "system", Schema: "runtime"}, wantCatalog: "system", wantSchema: "runtime"},
		{name: "catalog override clears schema", opts: QueryOptions{Catalog: "system"}, wantCatalog: "system", wantSchema: nil},
		{name: "schema override", opts: QueryOptions{Schema: "information_schema"}, wantCatalog: "memory", wantSchema: "information_schema"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Query(context.Background(), "SELECT current_catalog AS c, current_schema AS s", tt.opts)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if len(result.Rows) != 1 {
				t.Fatalf("Expected 1 row, got %d", len(result.Rows))
			}
			row := result.Rows[0]
			if row["c"] != tt.wantCatalog || row["s"] != tt.wantSchema {
				t.Errorf("session = %v/%v, want %v/%v", row["c"], row["s"], tt.wantCatalog, tt.wantSchema)
			}
		})
	}

	// An unqualified table name resolves against the session schema.
	if _, err := client.Query(context.Background(), "SELECT count(*) FROM tables", QueryOptions{Schema: "information_schema"}); err != nil {
		t.Errorf("unqualified table in session schema: %v", err)
	}
}

// TestIntegration_StatementTimeoutCancelsQuery covers #109: a statement the
// coordinator accepted and was still running when its timeout passed reports
// its query ID and a confirmed cancel, and the coordinator shows it canceled.
func TestIntegration_StatementTimeoutCancelsQuery(t *testing.T) {
	client := setupIntegrationClient(t)
	defer client.Close()

	ctx := context.Background()
	table := "memory.default.timeout_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	// Counting tpch.sf1000.lineitem runs for minutes on a single node.
	_, err := client.Query(ctx, "CREATE TABLE "+table+" AS SELECT count(*) AS c FROM tpch.sf1000.lineitem",
		QueryOptions{Timeout: 2 * time.Second})

	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("error %v (%T) is not a *QueryError", err, err)
	}
	if !qe.Accepted || qe.QueryID == "" || !qe.CancelConfirmed {
		t.Fatalf("QueryError %+v, want an accepted query with a confirmed cancel", qe)
	}
	class, ok := Classify(err)
	if !ok || class.StatementTimeout == nil || class.Category != CategoryClientInput || class.Retryable {
		t.Errorf("Classify = %+v, want a non-retryable client_input statement timeout", class)
	}

	// #nosec G202 -- the query ID comes from the coordinator, not user input
	res, err := client.Query(ctx, "SELECT state, error_code FROM system.runtime.queries WHERE query_id = '"+qe.QueryID+"'",
		QueryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 1 || res.Rows[0]["state"] != "FAILED" || res.Rows[0]["error_code"] != "USER_CANCELED" {
		t.Errorf("coordinator reports %v for %s, want FAILED with USER_CANCELED", res.Rows, qe.QueryID)
	}
}
