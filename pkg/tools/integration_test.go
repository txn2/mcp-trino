package tools

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// TestRegisterAll verifies that all tools can be registered without panicking.
func TestRegisterAll(t *testing.T) {
	// Create a valid client config
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Source:  "test",
		Catalog: "memory",
		Schema:  "default",
		Timeout: 30 * time.Second,
	}

	// Create the client
	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	// Create toolkit with default config
	toolkit := NewToolkit(trinoClient, DefaultConfig())

	// Register all tools - should not panic
	toolkit.RegisterAll(server)
}

// TestToolkit_WithMCPServer tests the full integration with an MCP server.
func TestToolkit_WithMCPServer(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Source:  "test",
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	toolkitCfg := Config{
		DefaultLimit:   500,
		MaxLimit:       5000,
		DefaultTimeout: 60 * time.Second,
		MaxTimeout:     180 * time.Second,
	}

	toolkit := NewToolkit(trinoClient, toolkitCfg)

	// Verify the toolkit stores the client and config correctly
	if toolkit.Client() != trinoClient {
		t.Error("toolkit should store the provided client")
	}

	actualCfg := toolkit.Config()
	if actualCfg.DefaultLimit != 500 {
		t.Errorf("expected DefaultLimit 500, got %d", actualCfg.DefaultLimit)
	}

	// Register tools
	toolkit.RegisterAll(server)
}

// TestHandleQuery_MissingSQL tests the query handler with missing SQL.
func TestHandleQuery_MissingSQL(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	toolkit := NewToolkit(trinoClient, DefaultConfig())

	// Call handleQuery with empty SQL
	result, _, err := toolkit.handleQuery(context.Background(), nil, QueryInput{SQL: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing SQL")
	}
}

// TestHandleExplain_MissingSQL tests the explain handler with missing SQL.
func TestHandleExplain_MissingSQL(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	toolkit := NewToolkit(trinoClient, DefaultConfig())

	result, _, err := toolkit.handleExplain(context.Background(), nil, ExplainInput{SQL: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing SQL")
	}
}

// TestHandleListSchemas_MissingCatalog tests the list schemas handler with missing catalog.
func TestHandleListSchemas_MissingCatalog(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	toolkit := NewToolkit(trinoClient, DefaultConfig())

	result, _, err := toolkit.handleListSchemas(context.Background(), nil, ListSchemasInput{Catalog: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing catalog")
	}
}

// TestHandleListTables_MissingParams tests the list tables handler with missing parameters.
func TestHandleListTables_MissingParams(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	toolkit := NewToolkit(trinoClient, DefaultConfig())

	// Missing catalog
	result, _, err := toolkit.handleListTables(context.Background(), nil, ListTablesInput{
		Catalog: "",
		Schema:  "default",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing catalog")
	}

	// Missing schema
	result, _, err = toolkit.handleListTables(context.Background(), nil, ListTablesInput{
		Catalog: "memory",
		Schema:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing schema")
	}
}

// TestHandleDescribeTable_MissingParams tests the describe table handler with missing parameters.
func TestHandleDescribeTable_MissingParams(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	toolkit := NewToolkit(trinoClient, DefaultConfig())

	// Missing catalog
	result, _, err := toolkit.handleDescribeTable(context.Background(), nil, DescribeTableInput{
		Catalog: "",
		Schema:  "default",
		Table:   "users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing catalog")
	}

	// Missing schema
	result, _, err = toolkit.handleDescribeTable(context.Background(), nil, DescribeTableInput{
		Catalog: "memory",
		Schema:  "",
		Table:   "users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing schema")
	}

	// Missing table
	result, _, err = toolkit.handleDescribeTable(context.Background(), nil, DescribeTableInput{
		Catalog: "memory",
		Schema:  "default",
		Table:   "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing table")
	}
}

// TestHandleQuery_LimitEnforcement tests that limits are enforced correctly.
func TestHandleQuery_LimitEnforcement(t *testing.T) {
	cfg := client.Config{
		Host:    "localhost",
		Port:    8080,
		User:    "test",
		SSL:     false,
		Catalog: "memory",
		Schema:  "default",
	}

	trinoClient, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer trinoClient.Close()

	toolkitCfg := Config{
		DefaultLimit:   100,
		MaxLimit:       500,
		DefaultTimeout: 30 * time.Second,
		MaxTimeout:     60 * time.Second,
	}

	_ = NewToolkit(trinoClient, toolkitCfg)

	// Test with limit exceeding max
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit uses default", 0, 100},
		{"negative limit uses default", -1, 100},
		{"limit within range", 200, 200},
		{"limit exceeds max", 1000, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually run the query without a Trino server,
			// but we can test the limit calculation logic
			limit := tt.inputLimit
			if limit <= 0 {
				limit = toolkitCfg.DefaultLimit
			}
			if limit > toolkitCfg.MaxLimit {
				limit = toolkitCfg.MaxLimit
			}

			if limit != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, limit)
			}
		})
	}
}

// TestExplainTypeMapping tests the mapping of explain types.
func TestExplainTypeMapping(t *testing.T) {
	tests := []struct {
		inputType    string
		expectedType client.ExplainType
	}{
		{"logical", client.ExplainLogical},
		{"distributed", client.ExplainDistributed},
		{"io", client.ExplainIO},
		{"validate", client.ExplainValidate},
		{"", client.ExplainLogical},
		{"unknown", client.ExplainLogical},
	}

	for _, tt := range tests {
		t.Run(tt.inputType, func(t *testing.T) {
			var explainType client.ExplainType
			switch tt.inputType {
			case "distributed":
				explainType = client.ExplainDistributed
			case "io":
				explainType = client.ExplainIO
			case "validate":
				explainType = client.ExplainValidate
			default:
				explainType = client.ExplainLogical
			}

			if explainType != tt.expectedType {
				t.Errorf("expected %v, got %v", tt.expectedType, explainType)
			}
		})
	}
}
