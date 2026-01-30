package tools

import (
	"context"

	"github.com/txn2/mcp-trino/pkg/client"
)

// MockTrinoClient is a test mock for TrinoClient interface.
type MockTrinoClient struct {
	// Query mock configuration
	QueryFunc   func(ctx context.Context, sql string, opts client.QueryOptions) (*client.QueryResult, error)
	QueryCalled bool
	QuerySQL    string
	QueryOpts   client.QueryOptions

	// Explain mock configuration
	ExplainFunc   func(ctx context.Context, sql string, explainType client.ExplainType) (*client.ExplainResult, error)
	ExplainCalled bool
	ExplainSQL    string
	ExplainType   client.ExplainType

	// ListCatalogs mock configuration
	ListCatalogsFunc   func(ctx context.Context) ([]string, error)
	ListCatalogsCalled bool

	// ListSchemas mock configuration
	ListSchemasFunc    func(ctx context.Context, catalog string) ([]string, error)
	ListSchemasCalled  bool
	ListSchemasCatalog string

	// ListTables mock configuration
	ListTablesFunc    func(ctx context.Context, catalog, schema string) ([]client.TableInfo, error)
	ListTablesCalled  bool
	ListTablesCatalog string
	ListTablesSchema  string

	// DescribeTable mock configuration
	DescribeTableFunc    func(ctx context.Context, catalog, schema, table string) (*client.TableInfo, error)
	DescribeTableCalled  bool
	DescribeTableCatalog string
	DescribeTableSchema  string
	DescribeTableTable   string
}

// NewMockTrinoClient creates a new mock client with default successful responses.
func NewMockTrinoClient() *MockTrinoClient {
	return &MockTrinoClient{
		QueryFunc: func(_ context.Context, _ string, _ client.QueryOptions) (*client.QueryResult, error) {
			return &client.QueryResult{
				Columns: []client.ColumnInfo{
					{Name: "id", Type: "INTEGER", Nullable: false},
					{Name: "name", Type: "VARCHAR", Nullable: true},
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
			}, nil
		},
		ExplainFunc: func(_ context.Context, _ string, explainType client.ExplainType) (*client.ExplainResult, error) {
			return &client.ExplainResult{
				Type: explainType,
				Plan: "Fragment 0 [SINGLE]\n    Output [id, name]\n    TableScan[memory:test_table]",
			}, nil
		},
		ListCatalogsFunc: func(_ context.Context) ([]string, error) {
			return []string{"memory", "hive", "postgresql"}, nil
		},
		ListSchemasFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"default", "information_schema", "test"}, nil
		},
		ListTablesFunc: func(_ context.Context, catalog, schema string) ([]client.TableInfo, error) {
			return []client.TableInfo{
				{Catalog: catalog, Schema: schema, Name: "users", Type: "TABLE"},
				{Catalog: catalog, Schema: schema, Name: "orders", Type: "TABLE"},
			}, nil
		},
		DescribeTableFunc: func(_ context.Context, catalog, schema, table string) (*client.TableInfo, error) {
			return &client.TableInfo{
				Catalog: catalog,
				Schema:  schema,
				Name:    table,
				Type:    "TABLE",
				Columns: []client.ColumnDef{
					{Name: "id", Type: "INTEGER", Nullable: "NO"},
					{Name: "name", Type: "VARCHAR(255)", Nullable: "YES"},
					{Name: "created_at", Type: "TIMESTAMP", Nullable: "YES", Comment: "Creation timestamp"},
				},
			}, nil
		},
	}
}

// Query implements TrinoClient.
func (m *MockTrinoClient) Query(ctx context.Context, sql string, opts client.QueryOptions) (*client.QueryResult, error) {
	m.QueryCalled = true
	m.QuerySQL = sql
	m.QueryOpts = opts
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, opts)
	}
	return nil, nil
}

// Explain implements TrinoClient.
func (m *MockTrinoClient) Explain(ctx context.Context, sql string, explainType client.ExplainType) (*client.ExplainResult, error) {
	m.ExplainCalled = true
	m.ExplainSQL = sql
	m.ExplainType = explainType
	if m.ExplainFunc != nil {
		return m.ExplainFunc(ctx, sql, explainType)
	}
	return nil, nil
}

// ListCatalogs implements TrinoClient.
func (m *MockTrinoClient) ListCatalogs(ctx context.Context) ([]string, error) {
	m.ListCatalogsCalled = true
	if m.ListCatalogsFunc != nil {
		return m.ListCatalogsFunc(ctx)
	}
	return nil, nil
}

// ListSchemas implements TrinoClient.
func (m *MockTrinoClient) ListSchemas(ctx context.Context, catalog string) ([]string, error) {
	m.ListSchemasCalled = true
	m.ListSchemasCatalog = catalog
	if m.ListSchemasFunc != nil {
		return m.ListSchemasFunc(ctx, catalog)
	}
	return nil, nil
}

// ListTables implements TrinoClient.
func (m *MockTrinoClient) ListTables(ctx context.Context, catalog, schema string) ([]client.TableInfo, error) {
	m.ListTablesCalled = true
	m.ListTablesCatalog = catalog
	m.ListTablesSchema = schema
	if m.ListTablesFunc != nil {
		return m.ListTablesFunc(ctx, catalog, schema)
	}
	return nil, nil
}

// DescribeTable implements TrinoClient.
func (m *MockTrinoClient) DescribeTable(ctx context.Context, catalog, schema, table string) (*client.TableInfo, error) {
	m.DescribeTableCalled = true
	m.DescribeTableCatalog = catalog
	m.DescribeTableSchema = schema
	m.DescribeTableTable = table
	if m.DescribeTableFunc != nil {
		return m.DescribeTableFunc(ctx, catalog, schema, table)
	}
	return nil, nil
}

// Ensure MockTrinoClient satisfies TrinoClient interface.
var _ TrinoClient = (*MockTrinoClient)(nil)
