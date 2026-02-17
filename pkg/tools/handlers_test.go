package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// TestHandleQuery_Success tests successful query execution.
func TestHandleQuery_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
		SQL:    "SELECT * FROM users",
		Limit:  100,
		Format: "json",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if !mock.QueryCalled {
		t.Error("Query was not called on mock client")
	}
	if mock.QuerySQL != "SELECT * FROM users" {
		t.Errorf("expected SQL 'SELECT * FROM users', got '%s'", mock.QuerySQL)
	}

	// Check result content
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}
}

// TestHandleQuery_CSVFormat tests CSV output format.
func TestHandleQuery_CSVFormat(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
		SQL:    "SELECT * FROM users",
		Format: "csv",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "id,name") {
		t.Error("expected CSV header")
	}
	if !strings.Contains(textContent.Text, "1,Alice") {
		t.Error("expected CSV data")
	}
}

// TestHandleQuery_MarkdownFormat tests markdown output format.
func TestHandleQuery_MarkdownFormat(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
		SQL:    "SELECT * FROM users",
		Format: "markdown",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "| id |") {
		t.Error("expected markdown table header")
	}
	if !strings.Contains(textContent.Text, "| --- |") {
		t.Error("expected markdown separator")
	}
}

// TestHandleQuery_Error tests query error handling.
func TestHandleQuery_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.QueryFunc = func(_ context.Context, _ string, _ client.QueryOptions) (*client.QueryResult, error) {
		return nil, errors.New("connection timeout")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
		SQL: "SELECT * FROM users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Query failed") {
		t.Error("expected error message")
	}
	if !strings.Contains(textContent.Text, "connection timeout") {
		t.Error("expected specific error")
	}
}

// TestHandleQuery_WithInterceptor tests SQL interception.
func TestHandleQuery_WithInterceptor(t *testing.T) {
	mock := NewMockTrinoClient()
	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " WHERE active = true", nil
	})
	toolkit := NewToolkit(mock, DefaultConfig(), WithQueryInterceptor(interceptor))

	_, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
		SQL: "SELECT * FROM users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.QuerySQL != "SELECT * FROM users WHERE active = true" {
		t.Errorf("expected intercepted SQL, got '%s'", mock.QuerySQL)
	}
}

// TestHandleQuery_InterceptorRejects tests interceptor rejection.
func TestHandleQuery_InterceptorRejects(t *testing.T) {
	mock := NewMockTrinoClient()
	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		if strings.Contains(strings.ToUpper(sql), "RESTRICTED_TABLE") {
			return "", errors.New("access to RESTRICTED_TABLE is not allowed")
		}
		return sql, nil
	})
	toolkit := NewToolkit(mock, DefaultConfig(), WithQueryInterceptor(interceptor))

	result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
		SQL: "SELECT * FROM RESTRICTED_TABLE",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Query rejected") {
		t.Error("expected rejection message")
	}
	if mock.QueryCalled {
		t.Error("Query should not be called when interceptor rejects")
	}
}

// TestHandleExplain_Success tests successful explain execution.
func TestHandleExplain_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExplain(context.Background(), nil, ExplainInput{
		SQL:  "SELECT * FROM users",
		Type: "logical",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.ExplainCalled {
		t.Error("Explain was not called")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Execution Plan") {
		t.Error("expected plan header")
	}
	if !strings.Contains(textContent.Text, "TableScan") {
		t.Error("expected plan content")
	}
}

// TestHandleExplain_AllTypes tests all explain types.
func TestHandleExplain_AllTypes(t *testing.T) {
	tests := []struct {
		inputType    string
		expectedType client.ExplainType
	}{
		{"logical", client.ExplainLogical},
		{"distributed", client.ExplainDistributed},
		{"io", client.ExplainIO},
		{"validate", client.ExplainValidate},
		{"", client.ExplainLogical}, // Default
	}

	for _, tt := range tests {
		t.Run("type_"+tt.inputType, func(t *testing.T) {
			mock := NewMockTrinoClient()
			toolkit := NewToolkit(mock, DefaultConfig())

			_, _, err := toolkit.handleExplain(context.Background(), nil, ExplainInput{
				SQL:  "SELECT 1",
				Type: tt.inputType,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mock.ExplainType != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, mock.ExplainType)
			}
		})
	}
}

// TestHandleExplain_Error tests explain error handling.
func TestHandleExplain_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.ExplainFunc = func(_ context.Context, _ string, _ client.ExplainType) (*client.ExplainResult, error) {
		return nil, errors.New("syntax error")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExplain(context.Background(), nil, ExplainInput{
		SQL: "SELECT * FORM users", // Intentional typo
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Explain failed") {
		t.Error("expected error message")
	}
}

// TestHandleExplain_WithInterceptor tests SQL interception for explain.
func TestHandleExplain_WithInterceptor(t *testing.T) {
	mock := NewMockTrinoClient()
	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " LIMIT 1", nil
	})
	toolkit := NewToolkit(mock, DefaultConfig(), WithQueryInterceptor(interceptor))

	_, _, err := toolkit.handleExplain(context.Background(), nil, ExplainInput{
		SQL: "SELECT * FROM users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.ExplainSQL != "SELECT * FROM users LIMIT 1" {
		t.Errorf("expected intercepted SQL, got '%s'", mock.ExplainSQL)
	}
}

// TestHandleListCatalogs_Success tests successful catalog listing.
func TestHandleListCatalogs_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListCatalogs(context.Background(), nil, ListCatalogsInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.ListCatalogsCalled {
		t.Error("ListCatalogs was not called")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "memory") {
		t.Error("expected 'memory' catalog")
	}
	if !strings.Contains(textContent.Text, "3 catalogs found") {
		t.Error("expected catalog count")
	}
}

// TestHandleListCatalogs_Error tests catalog listing error.
func TestHandleListCatalogs_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.ListCatalogsFunc = func(_ context.Context) ([]string, error) {
		return nil, errors.New("permission denied")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListCatalogs(context.Background(), nil, ListCatalogsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Failed to list catalogs") {
		t.Error("expected error message")
	}
}

// TestHandleListSchemas_Success tests successful schema listing.
func TestHandleListSchemas_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListSchemas(context.Background(), nil, ListSchemasInput{
		Catalog: "memory",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ListSchemasCatalog != "memory" {
		t.Errorf("expected catalog 'memory', got '%s'", mock.ListSchemasCatalog)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "default") {
		t.Error("expected 'default' schema")
	}
}

// TestHandleListSchemas_Error tests schema listing error.
func TestHandleListSchemas_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.ListSchemasFunc = func(_ context.Context, _ string) ([]string, error) {
		return nil, errors.New("catalog not found")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListSchemas(context.Background(), nil, ListSchemasInput{
		Catalog: "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Failed to list schemas") {
		t.Error("expected error message")
	}
}

// TestHandleListTables_Success tests successful table listing.
func TestHandleListTables_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListTables(context.Background(), nil, ListTablesInput{
		Catalog: "memory",
		Schema:  "default",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ListTablesCatalog != "memory" {
		t.Errorf("expected catalog 'memory', got '%s'", mock.ListTablesCatalog)
	}
	if mock.ListTablesSchema != "default" {
		t.Errorf("expected schema 'default', got '%s'", mock.ListTablesSchema)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "users") {
		t.Error("expected 'users' table")
	}
	if !strings.Contains(textContent.Text, "orders") {
		t.Error("expected 'orders' table")
	}
}

// TestHandleListTables_WithPattern tests table listing with pattern filter.
func TestHandleListTables_WithPattern(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListTables(context.Background(), nil, ListTablesInput{
		Catalog: "memory",
		Schema:  "default",
		Pattern: "%user%",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "users") {
		t.Error("expected 'users' table to match pattern")
	}
	if strings.Contains(textContent.Text, "orders") {
		t.Error("'orders' should not match '%user%' pattern")
	}
}

// TestHandleListTables_Error tests table listing error.
func TestHandleListTables_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.ListTablesFunc = func(_ context.Context, _, _ string) ([]client.TableInfo, error) {
		return nil, errors.New("schema not found")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleListTables(context.Background(), nil, ListTablesInput{
		Catalog: "memory",
		Schema:  "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Failed to list tables") {
		t.Error("expected error message")
	}
}

// TestHandleDescribeTable_Success tests successful table description.
func TestHandleDescribeTable_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleDescribeTable(context.Background(), nil, DescribeTableInput{
		Catalog: "memory",
		Schema:  "default",
		Table:   "users",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.DescribeTableTable != "users" {
		t.Errorf("expected table 'users', got '%s'", mock.DescribeTableTable)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "memory.default.users") {
		t.Error("expected full table name")
	}
	if !strings.Contains(textContent.Text, "id") {
		t.Error("expected 'id' column")
	}
	if !strings.Contains(textContent.Text, "INTEGER") {
		t.Error("expected column type")
	}
}

// TestHandleDescribeTable_WithSample tests table description with sample data.
func TestHandleDescribeTable_WithSample(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleDescribeTable(context.Background(), nil, DescribeTableInput{
		Catalog:       "memory",
		Schema:        "default",
		Table:         "users",
		IncludeSample: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Sample Data") {
		t.Error("expected sample data section")
	}
	// Query should be called for the sample
	if !mock.QueryCalled {
		t.Error("Query should be called for sample data")
	}
}

// TestHandleDescribeTable_Error tests table description error.
func TestHandleDescribeTable_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.DescribeTableFunc = func(_ context.Context, _, _, _ string) (*client.TableInfo, error) {
		return nil, errors.New("table not found")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleDescribeTable(context.Background(), nil, DescribeTableInput{
		Catalog: "memory",
		Schema:  "default",
		Table:   "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Failed to describe table") {
		t.Error("expected error message")
	}
}

// TestWrapHandler_WithMiddleware tests handler wrapping with middleware.
func TestWrapHandler_WithMiddleware(t *testing.T) {
	var beforeCalled, afterCalled bool
	var beforeOrder, afterOrder int
	callOrder := 0

	mw := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			beforeCalled = true
			callOrder++
			beforeOrder = callOrder
			return ctx, nil
		},
		AfterFn: func(_ context.Context, _ *ToolContext, result *mcp.CallToolResult, _ error) (*mcp.CallToolResult, error) {
			afterCalled = true
			callOrder++
			afterOrder = callOrder
			return result, nil
		},
	}

	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig(), WithMiddleware(mw))

	baseHandler := func(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		callOrder++
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
	}

	cfg := &toolConfig{}
	wrappedHandler := toolkit.wrapHandler(ToolQuery, baseHandler, cfg)
	_, _, err := wrappedHandler(context.Background(), nil, QueryInput{SQL: "SELECT 1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !beforeCalled {
		t.Error("Before hook not called")
	}
	if !afterCalled {
		t.Error("After hook not called")
	}
	if beforeOrder != 1 {
		t.Errorf("Before should be called first, got order %d", beforeOrder)
	}
	if afterOrder != 3 {
		t.Errorf("After should be called last, got order %d", afterOrder)
	}
}

// TestWrapHandler_MiddlewareError tests middleware error handling.
func TestWrapHandler_MiddlewareError(t *testing.T) {
	mw := MiddlewareFunc{
		BeforeFn: func(_ context.Context, _ *ToolContext) (context.Context, error) {
			return nil, errors.New("auth failed")
		},
	}

	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig(), WithMiddleware(mw))

	handlerCalled := false
	baseHandler := func(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		handlerCalled = true
		return &mcp.CallToolResult{}, nil, nil
	}

	wrappedHandler := toolkit.wrapHandler(ToolQuery, baseHandler, nil)
	result, _, err := wrappedHandler(context.Background(), nil, QueryInput{SQL: "SELECT 1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if handlerCalled {
		t.Error("Handler should not be called when middleware fails")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "middleware error") {
		t.Error("expected middleware error message")
	}
}

// TestWrapHandler_WithTransformer tests result transformation.
func TestWrapHandler_WithTransformer(t *testing.T) {
	transformer := ResultTransformerFunc(func(_ context.Context, _ ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp.TextContent); ok {
				tc.Text = "[TRANSFORMED] " + tc.Text
			}
		}
		return result, nil
	})

	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig(), WithResultTransformer(transformer))

	baseHandler := func(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "original"}}}, nil, nil
	}

	wrappedHandler := toolkit.wrapHandler(ToolQuery, baseHandler, nil)
	result, _, err := wrappedHandler(context.Background(), nil, QueryInput{SQL: "SELECT 1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "[TRANSFORMED]") {
		t.Error("expected transformed result")
	}
}

// TestInterceptSQL tests SQL interception chain.
func TestInterceptSQL(t *testing.T) {
	i1 := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " /* i1 */", nil
	})
	i2 := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " /* i2 */", nil
	})

	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig(), WithQueryInterceptor(i1), WithQueryInterceptor(i2))

	result, err := toolkit.InterceptSQL(context.Background(), "SELECT 1", ToolQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "SELECT 1 /* i1 */ /* i2 */" {
		t.Errorf("unexpected result: %s", result)
	}
}

// TestInterceptSQL_NoInterceptors tests InterceptSQL with no interceptors.
func TestInterceptSQL_NoInterceptors(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, err := toolkit.InterceptSQL(context.Background(), "SELECT 1", ToolQuery)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "SELECT 1" {
		t.Errorf("expected unchanged SQL, got: %s", result)
	}
}

// TestInterceptSQL_Error tests interceptor error handling.
func TestInterceptSQL_Error(t *testing.T) {
	interceptor := QueryInterceptorFunc(func(_ context.Context, _ string, _ ToolName) (string, error) {
		return "", errors.New("forbidden")
	})

	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig(), WithQueryInterceptor(interceptor))

	_, err := toolkit.InterceptSQL(context.Background(), "DROP TABLE users", ToolQuery)
	if err == nil {
		t.Error("expected error")
	}
}

// TestRegisterWith tests per-registration options.
func TestRegisterWith(t *testing.T) {
	middlewareCalled := false
	mw := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			middlewareCalled = true
			return ctx, nil
		},
	}

	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	toolkit.RegisterWith(server, ToolQuery, WithPerToolMiddleware(mw))

	// Verify registration
	if !toolkit.registeredTools[ToolQuery] {
		t.Error("ToolQuery should be registered")
	}

	// The middleware is attached but won't be called until the tool is invoked
	// This test verifies registration works without error
	_ = middlewareCalled // Used to attach middleware
}

// TestTextResult tests the TextResult helper.
func TestTextResult(t *testing.T) {
	result := TextResult("hello world")

	if len(result.Content) != 1 {
		t.Fatal("expected 1 content item")
	}

	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", tc.Text)
	}
}

// TestQueryTimeoutEnforcement tests timeout boundary enforcement.
func TestQueryTimeoutEnforcement(t *testing.T) {
	mock := NewMockTrinoClient()
	cfg := Config{
		DefaultLimit:   1000,
		MaxLimit:       10000,
		DefaultTimeout: 60 * time.Second,
		MaxTimeout:     120 * time.Second,
	}
	toolkit := NewToolkit(mock, cfg)

	tests := []struct {
		name            string
		inputTimeout    int
		expectedTimeout time.Duration
	}{
		{"zero uses default", 0, 60 * time.Second},
		{"within range", 30, 30 * time.Second},
		{"exceeds max", 200, 120 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
				SQL:            "SELECT 1",
				TimeoutSeconds: tt.inputTimeout,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// We can't directly check the timeout applied, but we verify no panics
			// and that the query was called
			if !mock.QueryCalled {
				t.Error("Query should be called")
			}
			mock.QueryCalled = false // Reset for next iteration
		})
	}
}

// TestHandleQuery_ReadOnlyEnforcement tests that trino_query blocks write SQL.
func TestHandleQuery_ReadOnlyEnforcement(t *testing.T) {
	writeSQLs := []struct {
		name string
		sql  string
	}{
		{"INSERT", "INSERT INTO users VALUES (1, 'Alice')"},
		{"UPDATE", "UPDATE users SET name = 'Bob' WHERE id = 1"},
		{"DELETE", "DELETE FROM users WHERE id = 1"},
		{"DROP", "DROP TABLE users"},
		{"CREATE", "CREATE TABLE new_table (id INT)"},
		{"ALTER", "ALTER TABLE users ADD COLUMN email VARCHAR"},
		{"TRUNCATE", "TRUNCATE TABLE users"},
		{"GRANT", "GRANT SELECT ON users TO analyst"},
		{"REVOKE", "REVOKE SELECT ON users FROM analyst"},
		{"MERGE", "MERGE INTO target USING source ON target.id = source.id"},
		{"lowercase", "insert into users values (1, 'Alice')"},
		{"leading whitespace", "  INSERT INTO users VALUES (1)"},
		{"with comment", "-- comment\nINSERT INTO users VALUES (1)"},
	}

	for _, tt := range writeSQLs {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockTrinoClient()
			toolkit := NewToolkit(mock, DefaultConfig())

			result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
				SQL: tt.sql,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Error("expected error result for write SQL in trino_query")
			}
			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("expected TextContent")
			}
			if !strings.Contains(textContent.Text, "read-only") {
				t.Errorf("expected read-only error message, got: %s", textContent.Text)
			}
			if !strings.Contains(textContent.Text, "trino_execute") {
				t.Errorf("expected reference to trino_execute, got: %s", textContent.Text)
			}
			if mock.QueryCalled {
				t.Error("Query should not be called for write SQL in trino_query")
			}
		})
	}
}

// TestHandleQuery_AllowsReadSQL tests that trino_query allows read operations.
func TestHandleQuery_AllowsReadSQL(t *testing.T) {
	readSQLs := []struct {
		name string
		sql  string
	}{
		{"SELECT", "SELECT * FROM users"},
		{"SHOW", "SHOW TABLES"},
		{"DESCRIBE", "DESCRIBE users"},
		{"EXPLAIN", "EXPLAIN SELECT * FROM users"},
		{"WITH", "WITH cte AS (SELECT 1) SELECT * FROM cte"},
		{"select lowercase", "select id from users"},
	}

	for _, tt := range readSQLs {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockTrinoClient()
			toolkit := NewToolkit(mock, DefaultConfig())

			result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{
				SQL: tt.sql,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.IsError {
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if ok {
					t.Errorf("expected success for read SQL %q, got error: %s", tt.sql, textContent.Text)
				}
			}
			if !mock.QueryCalled {
				t.Errorf("Query should be called for read SQL %q", tt.sql)
			}
		})
	}
}

// TestHandleExecute_Success tests successful execute execution.
func TestHandleExecute_Success(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL:    "INSERT INTO users VALUES (3, 'Charlie')",
		Format: "json",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.IsError {
		t.Error("expected success result")
	}
	if !mock.QueryCalled {
		t.Error("Query was not called on mock client")
	}
	if mock.QuerySQL != "INSERT INTO users VALUES (3, 'Charlie')" {
		t.Errorf("expected SQL 'INSERT INTO users VALUES (3, 'Charlie')', got '%s'", mock.QuerySQL)
	}
}

// TestHandleExecute_AllowsWriteSQL tests that trino_execute allows write operations.
func TestHandleExecute_AllowsWriteSQL(t *testing.T) {
	writeSQLs := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'Bob'",
		"DELETE FROM users WHERE id = 1",
		"DROP TABLE temp_table",
		"CREATE TABLE new_table (id INT)",
	}

	for _, sql := range writeSQLs {
		t.Run(sql[:6], func(t *testing.T) {
			mock := NewMockTrinoClient()
			toolkit := NewToolkit(mock, DefaultConfig())

			result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{SQL: sql})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.IsError {
				t.Errorf("trino_execute should allow write SQL: %s", sql)
			}
			if !mock.QueryCalled {
				t.Error("Query should be called")
			}
		})
	}
}

// TestHandleExecute_AllowsReadSQL tests that trino_execute also allows read operations.
func TestHandleExecute_AllowsReadSQL(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL: "SELECT * FROM users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("trino_execute should allow read SQL too")
	}
}

// TestHandleExecute_EmptySQL tests error for empty SQL.
func TestHandleExecute_EmptySQL(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{SQL: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for empty SQL")
	}
}

// TestHandleExecute_Error tests execute error handling.
func TestHandleExecute_Error(t *testing.T) {
	mock := NewMockTrinoClient()
	mock.QueryFunc = func(_ context.Context, _ string, _ client.QueryOptions) (*client.QueryResult, error) {
		return nil, errors.New("table not found")
	}
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL: "INSERT INTO nonexistent VALUES (1)",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Execution failed") {
		t.Error("expected error message")
	}
}

// TestHandleExecute_WithInterceptor tests SQL interception for execute.
func TestHandleExecute_WithInterceptor(t *testing.T) {
	mock := NewMockTrinoClient()
	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql + " /* audited */", nil
	})
	toolkit := NewToolkit(mock, DefaultConfig(), WithQueryInterceptor(interceptor))

	_, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL: "INSERT INTO users VALUES (1)",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.QuerySQL != "INSERT INTO users VALUES (1) /* audited */" {
		t.Errorf("expected intercepted SQL, got '%s'", mock.QuerySQL)
	}
}

// TestHandleExecute_CSVFormat tests CSV output format for execute.
func TestHandleExecute_CSVFormat(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL:    "SELECT * FROM users",
		Format: "csv",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "id,name") {
		t.Error("expected CSV header")
	}
}

// TestHandleExecute_MarkdownFormat tests markdown output format for execute.
func TestHandleExecute_MarkdownFormat(t *testing.T) {
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL:    "SELECT * FROM users",
		Format: "markdown",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if !strings.Contains(textContent.Text, "| id |") {
		t.Error("expected markdown table header")
	}
}

// TestHandleExecute_ConnectionError tests connection error handling.
func TestHandleExecute_ConnectionError(t *testing.T) {
	toolkit := NewToolkit(nil, DefaultConfig()) // nil client

	result, _, err := toolkit.handleExecute(context.Background(), nil, ExecuteInput{
		SQL: "SELECT 1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nil client")
	}
}

// TestRegisteredToolInvocation exercises the full registration + MCP dispatch path
// for all tools, covering the typed-output wrapper closures in register*Tool functions.
func TestRegisteredToolInvocation(t *testing.T) {
	ctx := context.Background()
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	toolkit.RegisterAll(server)

	// Connect client and server via in-memory transport
	t1, t2 := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	clientSession, err := mcpClient.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer clientSession.Close()

	tests := []struct {
		name   string
		params mcp.CallToolParams
	}{
		{
			name: "trino_query",
			params: mcp.CallToolParams{
				Name:      "trino_query",
				Arguments: map[string]any{"sql": "SELECT 1"},
			},
		},
		{
			name: "trino_execute",
			params: mcp.CallToolParams{
				Name:      "trino_execute",
				Arguments: map[string]any{"sql": "INSERT INTO users VALUES (1, 'Test')"},
			},
		},
		{
			name: "trino_explain",
			params: mcp.CallToolParams{
				Name:      "trino_explain",
				Arguments: map[string]any{"sql": "SELECT 1"},
			},
		},
		{
			name: "trino_list_catalogs",
			params: mcp.CallToolParams{
				Name: "trino_list_catalogs",
			},
		},
		{
			name: "trino_list_schemas",
			params: mcp.CallToolParams{
				Name:      "trino_list_schemas",
				Arguments: map[string]any{"catalog": "memory"},
			},
		},
		{
			name: "trino_list_tables",
			params: mcp.CallToolParams{
				Name:      "trino_list_tables",
				Arguments: map[string]any{"catalog": "memory", "schema": "default"},
			},
		},
		{
			name: "trino_describe_table",
			params: mcp.CallToolParams{
				Name:      "trino_describe_table",
				Arguments: map[string]any{"catalog": "memory", "schema": "default", "table": "users"},
			},
		},
		{
			name: "trino_list_connections",
			params: mcp.CallToolParams{
				Name: "trino_list_connections",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := clientSession.CallTool(ctx, &tt.params)
			if err != nil {
				t.Fatalf("CallTool(%s) error: %v", tt.name, err)
			}
			if result == nil {
				t.Fatalf("CallTool(%s) returned nil result", tt.name)
			}
			if len(result.Content) == 0 {
				t.Errorf("CallTool(%s) returned empty content", tt.name)
			}
		})
	}
}
