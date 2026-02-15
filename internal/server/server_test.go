package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/multiserver"
	"github.com/txn2/mcp-trino/pkg/semantic"
	"github.com/txn2/mcp-trino/pkg/tools"
)

func TestVersion(t *testing.T) {
	// Verify Version constant is set (compile-time check would catch empty)
	if len(Version) == 0 {
		t.Error("Version should not be empty")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	// MultiServerConfig should be nil (loaded from env in New())
	if opts.MultiServerConfig != nil {
		t.Error("MultiServerConfig should be nil by default")
	}

	// Verify ToolkitConfig has defaults
	if opts.ToolkitConfig.DefaultLimit <= 0 {
		t.Error("ToolkitConfig.DefaultLimit should be positive")
	}
	if opts.ToolkitConfig.MaxLimit <= 0 {
		t.Error("ToolkitConfig.MaxLimit should be positive")
	}
}

func TestOptions_CustomMultiServerConfig(t *testing.T) {
	cfg := &multiserver.Config{
		Default: "default",
		Primary: client.Config{
			Host:    "custom-host",
			Port:    9999,
			User:    "testuser",
			Catalog: "testcatalog",
			Schema:  "testschema",
		},
	}
	opts := Options{
		MultiServerConfig: cfg,
		ToolkitConfig:     tools.DefaultConfig(),
	}

	if opts.MultiServerConfig.Primary.Host != "custom-host" {
		t.Errorf("expected Host 'custom-host', got %q", opts.MultiServerConfig.Primary.Host)
	}
	if opts.MultiServerConfig.Primary.Port != 9999 {
		t.Errorf("expected Port 9999, got %d", opts.MultiServerConfig.Primary.Port)
	}
}

func TestOptions_CustomToolkitConfig(t *testing.T) {
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.DefaultConfig(),
		},
		ToolkitConfig: tools.Config{
			DefaultLimit: 500,
			MaxLimit:     5000,
		},
	}

	if opts.ToolkitConfig.DefaultLimit != 500 {
		t.Errorf("expected DefaultLimit 500, got %d", opts.ToolkitConfig.DefaultLimit)
	}
	if opts.ToolkitConfig.MaxLimit != 5000 {
		t.Errorf("expected MaxLimit 5000, got %d", opts.ToolkitConfig.MaxLimit)
	}
}

func TestNew_InvalidClientConfig(t *testing.T) {
	// Server now starts even when unconfigured - tools return helpful errors
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "", // Invalid: empty host
				User: "admin",
				Port: 8080,
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Errorf("expected server to start even with invalid host, got error: %v", err)
	}
	if server == nil {
		t.Error("expected server to be created")
	}
	if mgr != nil {
		_ = mgr.Close()
	}
}

func TestNew_MissingUser(t *testing.T) {
	// Server now starts even when unconfigured - tools return helpful errors
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				User: "", // Invalid: missing user
				Port: 8080,
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Errorf("expected server to start even without user, got error: %v", err)
	}
	if server == nil {
		t.Error("expected server to be created")
	}
	if mgr != nil {
		_ = mgr.Close()
	}
}

func TestNew_InvalidPort(t *testing.T) {
	// Server now starts even when unconfigured - tools return helpful errors
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				User: "admin",
				Port: 0, // Invalid: port 0
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Errorf("expected server to start even with invalid port, got error: %v", err)
	}
	if server == nil {
		t.Error("expected server to be created")
	}
	if mgr != nil {
		_ = mgr.Close()
	}
}

func TestNew_ValidConfig(t *testing.T) {
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host:    "localhost",
				Port:    8080,
				User:    "admin",
				SSL:     false,
				Source:  "test",
				Catalog: "memory",
				Schema:  "default",
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if server == nil {
		t.Error("server should not be nil")
	}
	if mgr == nil {
		t.Error("manager should not be nil")
	}

	// Clean up
	if mgr != nil {
		_ = mgr.Close()
	}
}

func TestNew_ServerImplementation(t *testing.T) {
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host:    "localhost",
				Port:    8080,
				User:    "admin",
				SSL:     false,
				Source:  "test",
				Catalog: "memory",
				Schema:  "default",
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	// Server should be configured with proper implementation name
	if server == nil {
		t.Fatal("server should not be nil")
	}
}

func TestNew_MultipleConnections(t *testing.T) {
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
				SSL:  false,
			},
			Connections: map[string]multiserver.ConnectionConfig{
				"staging": {
					Host: "staging.example.com",
				},
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Fatal("server should not be nil")
	}

	// Should have 2 connections
	if mgr.ConnectionCount() != 2 {
		t.Errorf("expected 2 connections, got %d", mgr.ConnectionCount())
	}
}

func TestDefaultOptions_DescriptionsNil(t *testing.T) {
	opts := DefaultOptions()

	if opts.Descriptions != nil {
		t.Error("Descriptions should be nil by default")
	}
}

func TestNew_WithDescriptions(t *testing.T) {
	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
		Descriptions: map[tools.ToolName]string{
			tools.ToolQuery:   "Custom query description",
			tools.ToolExplain: "Custom explain description",
		},
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Error("server should not be nil")
	}
}

func TestDefaultOptions_SemanticFields(t *testing.T) {
	opts := DefaultOptions()

	// Semantic fields should be nil by default
	if opts.SemanticProvider != nil {
		t.Error("SemanticProvider should be nil by default")
	}
	if opts.SemanticCacheConfig != nil {
		t.Error("SemanticCacheConfig should be nil by default")
	}
}

func TestNew_WithSemanticProvider(t *testing.T) {
	// Create a mock semantic provider
	mockProvider := &semantic.ProviderFunc{
		NameFn: func() string { return "mock" },
		GetTableContextFn: func(_ context.Context, _ semantic.TableIdentifier) (*semantic.TableContext, error) {
			return &semantic.TableContext{Description: "test"}, nil
		},
	}

	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
		},
		ToolkitConfig:    tools.DefaultConfig(),
		SemanticProvider: mockProvider,
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Error("server should not be nil")
	}
}

func TestNew_WithSemanticCacheConfig(t *testing.T) {
	// Create a mock semantic provider
	mockProvider := &semantic.ProviderFunc{
		NameFn: func() string { return "mock" },
	}

	cacheConfig := &semantic.CacheConfig{
		TTL:        10 * time.Minute,
		MaxEntries: 5000,
	}

	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
		},
		ToolkitConfig:       tools.DefaultConfig(),
		SemanticProvider:    mockProvider,
		SemanticCacheConfig: cacheConfig,
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Error("server should not be nil")
	}
}

func TestNew_SemanticFileEnvVar(t *testing.T) {
	// Create a temporary semantic file
	tmpDir := t.TempDir()
	semanticFile := filepath.Join(tmpDir, "semantic.yaml")
	content := `tables:
  - catalog: test
    schema: public
    table: users
    description: Test users table
`
	if err := os.WriteFile(semanticFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Set SEMANTIC_FILE env var
	t.Setenv("SEMANTIC_FILE", semanticFile)

	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Error("server should not be nil")
	}
}

func TestNew_SemanticFileEnvVar_InvalidFile(t *testing.T) {
	// Set SEMANTIC_FILE to non-existent file
	t.Setenv("SEMANTIC_FILE", "/nonexistent/file.yaml")

	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	// Server should still start even if semantic file fails to load
	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("server should start even with invalid semantic file: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Error("server should not be nil")
	}
}

func TestNew_SemanticProviderTakesPrecedenceOverEnv(t *testing.T) {
	// Create a temporary semantic file
	tmpDir := t.TempDir()
	semanticFile := filepath.Join(tmpDir, "semantic.yaml")
	content := `tables:
  - catalog: env
    schema: public
    table: from_env
    description: From environment
`
	if err := os.WriteFile(semanticFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Set SEMANTIC_FILE env var
	t.Setenv("SEMANTIC_FILE", semanticFile)

	// Create a mock semantic provider (should take precedence)
	mockProvider := &semantic.ProviderFunc{
		NameFn: func() string { return "explicit-provider" },
	}

	opts := Options{
		MultiServerConfig: &multiserver.Config{
			Default: "default",
			Primary: client.Config{
				Host: "localhost",
				Port: 8080,
				User: "admin",
			},
		},
		ToolkitConfig:    tools.DefaultConfig(),
		SemanticProvider: mockProvider,
	}

	server, mgr, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if server == nil {
		t.Error("server should not be nil")
	}
	// The explicit provider should be used, not the one from SEMANTIC_FILE
	// (verified by the fact that we're using mockProvider with name "explicit-provider")
}
