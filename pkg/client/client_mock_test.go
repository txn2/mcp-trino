package client

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestClient_Query(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		Catalog: "memory",
		Schema:  "default",
		Timeout: 30 * time.Second,
	}

	client := NewWithDB(db, cfg)

	t.Run("successful query", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "Alice").
			AddRow(2, "Bob")

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		result, err := client.Query(context.Background(), "SELECT id, name FROM users", DefaultQueryOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(result.Columns))
		}
		if len(result.Rows) != 2 {
			t.Errorf("expected 2 rows, got %d", len(result.Rows))
		}
		if result.Stats.RowCount != 2 {
			t.Errorf("expected RowCount 2, got %d", result.Stats.RowCount)
		}
	})

	t.Run("query with limit", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).
			AddRow(1).
			AddRow(2).
			AddRow(3)

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		opts := QueryOptions{Limit: 2}
		result, err := client.Query(context.Background(), "SELECT id FROM test", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Rows) != 2 {
			t.Errorf("expected 2 rows (limited), got %d", len(result.Rows))
		}
		if !result.Stats.Truncated {
			t.Error("expected Truncated to be true")
		}
	})

	t.Run("empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name"})
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		result, err := client.Query(context.Background(), "SELECT id, name FROM empty", DefaultQueryOptions())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(result.Rows))
		}
		if result.Stats.RowCount != 0 {
			t.Errorf("expected RowCount 0, got %d", result.Stats.RowCount)
		}
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT").WillReturnError(sqlmock.ErrCancelled)

		_, err := client.Query(context.Background(), "SELECT * FROM error", DefaultQueryOptions())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestClient_Query_DefaultLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		Timeout: 30 * time.Second,
	}

	client := NewWithDB(db, cfg)

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	// Test with zero limit (should use default 1000)
	opts := QueryOptions{Limit: 0}
	result, err := client.Query(context.Background(), "SELECT id FROM test", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Stats.LimitApplied != 1000 {
		t.Errorf("expected default limit 1000, got %d", result.Stats.LimitApplied)
	}
}

func TestClient_Query_CustomTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		Timeout: 60 * time.Second,
	}

	client := NewWithDB(db, cfg)

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	opts := QueryOptions{
		Timeout: 10 * time.Second,
	}
	_, err = client.Query(context.Background(), "SELECT id FROM test", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Explain(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		Timeout: 30 * time.Second,
	}

	client := NewWithDB(db, cfg)

	t.Run("logical explain", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"plan"}).
			AddRow("- Output").
			AddRow("  - TableScan")

		mock.ExpectQuery("EXPLAIN \\(TYPE LOGICAL\\)").WillReturnRows(rows)

		result, err := client.Explain(context.Background(), "SELECT 1", ExplainLogical)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Type != ExplainLogical {
			t.Errorf("expected type LOGICAL, got %s", result.Type)
		}
		if result.Plan != "- Output\n  - TableScan" {
			t.Errorf("unexpected plan: %q", result.Plan)
		}
	})

	t.Run("distributed explain", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"plan"}).AddRow("Fragment 0")
		mock.ExpectQuery("EXPLAIN \\(TYPE DISTRIBUTED\\)").WillReturnRows(rows)

		result, err := client.Explain(context.Background(), "SELECT 1", ExplainDistributed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Type != ExplainDistributed {
			t.Errorf("expected type DISTRIBUTED, got %s", result.Type)
		}
	})

	t.Run("io explain", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"plan"}).AddRow("IO estimate")
		mock.ExpectQuery("EXPLAIN \\(TYPE IO\\)").WillReturnRows(rows)

		result, err := client.Explain(context.Background(), "SELECT 1", ExplainIO)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Type != ExplainIO {
			t.Errorf("expected type IO, got %s", result.Type)
		}
	})

	t.Run("validate explain", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"plan"}).AddRow("Valid")
		mock.ExpectQuery("EXPLAIN \\(TYPE VALIDATE\\)").WillReturnRows(rows)

		result, err := client.Explain(context.Background(), "SELECT 1", ExplainValidate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Type != ExplainValidate {
			t.Errorf("expected type VALIDATE, got %s", result.Type)
		}
	})

	t.Run("default to logical", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"plan"}).AddRow("Plan")
		mock.ExpectQuery("EXPLAIN \\(TYPE LOGICAL\\)").WillReturnRows(rows)

		result, err := client.Explain(context.Background(), "SELECT 1", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Type != ExplainLogical {
			t.Errorf("expected default type LOGICAL, got %s", result.Type)
		}
	})

	t.Run("explain error", func(t *testing.T) {
		mock.ExpectQuery("EXPLAIN").WillReturnError(sqlmock.ErrCancelled)

		_, err := client.Explain(context.Background(), "INVALID SQL", ExplainLogical)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestClient_ListCatalogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{Host: "localhost", Port: 8080, User: "test"}
	client := NewWithDB(db, cfg)

	t.Run("successful list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Catalog"}).
			AddRow("hive").
			AddRow("memory").
			AddRow("system")

		mock.ExpectQuery("SHOW CATALOGS").WillReturnRows(rows)

		catalogs, err := client.ListCatalogs(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(catalogs) != 3 {
			t.Errorf("expected 3 catalogs, got %d", len(catalogs))
		}
		if catalogs[0] != "hive" {
			t.Errorf("expected first catalog 'hive', got %q", catalogs[0])
		}
	})

	t.Run("empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Catalog"})
		mock.ExpectQuery("SHOW CATALOGS").WillReturnRows(rows)

		catalogs, err := client.ListCatalogs(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(catalogs) != 0 {
			t.Errorf("expected 0 catalogs, got %d", len(catalogs))
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SHOW CATALOGS").WillReturnError(sqlmock.ErrCancelled)

		_, err := client.ListCatalogs(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestClient_ListSchemas(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{Host: "localhost", Port: 8080, User: "test", Catalog: "default_catalog"}
	client := NewWithDB(db, cfg)

	t.Run("successful list with catalog", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Schema"}).
			AddRow("default").
			AddRow("information_schema")

		mock.ExpectQuery("SHOW SCHEMAS FROM hive").WillReturnRows(rows)

		schemas, err := client.ListSchemas(context.Background(), "hive")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(schemas) != 2 {
			t.Errorf("expected 2 schemas, got %d", len(schemas))
		}
	})

	t.Run("uses default catalog when empty", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Schema"}).AddRow("default")
		mock.ExpectQuery("SHOW SCHEMAS FROM default_catalog").WillReturnRows(rows)

		schemas, err := client.ListSchemas(context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(schemas) != 1 {
			t.Errorf("expected 1 schema, got %d", len(schemas))
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SHOW SCHEMAS").WillReturnError(sqlmock.ErrCancelled)

		_, err := client.ListSchemas(context.Background(), "hive")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestClient_ListTables(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		Catalog: "default_catalog",
		Schema:  "default_schema",
	}
	client := NewWithDB(db, cfg)

	t.Run("successful list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Table"}).
			AddRow("users").
			AddRow("orders").
			AddRow("products")

		mock.ExpectQuery("SHOW TABLES FROM hive.sales").WillReturnRows(rows)

		tables, err := client.ListTables(context.Background(), "hive", "sales")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tables) != 3 {
			t.Errorf("expected 3 tables, got %d", len(tables))
		}
		if tables[0].Name != "users" {
			t.Errorf("expected first table 'users', got %q", tables[0].Name)
		}
		if tables[0].Catalog != "hive" {
			t.Errorf("expected catalog 'hive', got %q", tables[0].Catalog)
		}
		if tables[0].Schema != "sales" {
			t.Errorf("expected schema 'sales', got %q", tables[0].Schema)
		}
	})

	t.Run("uses defaults when empty", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Table"}).AddRow("test")
		mock.ExpectQuery("SHOW TABLES FROM default_catalog.default_schema").WillReturnRows(rows)

		tables, err := client.ListTables(context.Background(), "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tables) != 1 {
			t.Errorf("expected 1 table, got %d", len(tables))
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SHOW TABLES").WillReturnError(sqlmock.ErrCancelled)

		_, err := client.ListTables(context.Background(), "hive", "sales")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestClient_DescribeTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		Catalog: "default_catalog",
		Schema:  "default_schema",
	}
	client := NewWithDB(db, cfg)

	t.Run("successful describe", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Column", "Type", "Extra", "Comment"}).
			AddRow("id", "bigint", "NOT NULL", "Primary key").
			AddRow("name", "varchar", "", "User name").
			AddRow("email", "varchar", "NULL", nil)

		mock.ExpectQuery("DESCRIBE hive.sales.users").WillReturnRows(rows)

		info, err := client.DescribeTable(context.Background(), "hive", "sales", "users")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.Catalog != "hive" {
			t.Errorf("expected catalog 'hive', got %q", info.Catalog)
		}
		if info.Schema != "sales" {
			t.Errorf("expected schema 'sales', got %q", info.Schema)
		}
		if info.Name != "users" {
			t.Errorf("expected table 'users', got %q", info.Name)
		}
		if len(info.Columns) != 3 {
			t.Errorf("expected 3 columns, got %d", len(info.Columns))
		}
		if info.Columns[0].Name != "id" {
			t.Errorf("expected first column 'id', got %q", info.Columns[0].Name)
		}
		if info.Columns[0].Comment != "Primary key" {
			t.Errorf("expected comment 'Primary key', got %q", info.Columns[0].Comment)
		}
	})

	t.Run("uses defaults when empty", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Column", "Type", "Extra", "Comment"}).
			AddRow("id", "int", "", nil)

		mock.ExpectQuery("DESCRIBE default_catalog.default_schema.test").WillReturnRows(rows)

		info, err := client.DescribeTable(context.Background(), "", "", "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info.Catalog != "default_catalog" {
			t.Errorf("expected default catalog, got %q", info.Catalog)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("DESCRIBE").WillReturnError(sqlmock.ErrCancelled)

		_, err := client.DescribeTable(context.Background(), "hive", "sales", "nonexistent")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestNewWithDB(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Host:    "testhost",
		Port:    9090,
		User:    "testuser",
		Catalog: "testcatalog",
	}

	client := NewWithDB(db, cfg)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	returnedCfg := client.Config()
	if returnedCfg.Host != "testhost" {
		t.Errorf("expected Host 'testhost', got %q", returnedCfg.Host)
	}
	if returnedCfg.Catalog != "testcatalog" {
		t.Errorf("expected Catalog 'testcatalog', got %q", returnedCfg.Catalog)
	}
}
