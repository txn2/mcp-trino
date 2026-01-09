package tools

import (
	"context"

	"github.com/txn2/mcp-trino/pkg/client"
)

// TrinoClient defines the interface for Trino operations used by the toolkit.
// This interface is satisfied by *client.Client and allows for mock implementations in tests.
type TrinoClient interface {
	// Query executes a SQL query and returns the results.
	Query(ctx context.Context, sql string, opts client.QueryOptions) (*client.QueryResult, error)

	// Explain returns the execution plan for a query.
	Explain(ctx context.Context, sql string, explainType client.ExplainType) (*client.ExplainResult, error)

	// ListCatalogs returns available catalogs.
	ListCatalogs(ctx context.Context) ([]string, error)

	// ListSchemas returns schemas in the given catalog.
	ListSchemas(ctx context.Context, catalog string) ([]string, error)

	// ListTables returns tables in the given catalog and schema.
	ListTables(ctx context.Context, catalog, schema string) ([]client.TableInfo, error)

	// DescribeTable returns detailed information about a table.
	DescribeTable(ctx context.Context, catalog, schema, table string) (*client.TableInfo, error)
}

// Ensure *client.Client satisfies TrinoClient interface.
var _ TrinoClient = (*client.Client)(nil)
