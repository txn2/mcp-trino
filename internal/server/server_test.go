package server

import (
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/multiserver"
	"github.com/txn2/mcp-trino/pkg/tools"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Version != "0.1.0" {
		t.Errorf("expected Version '0.1.0', got %q", Version)
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
		mgr.Close()
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
	defer mgr.Close()

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
	defer mgr.Close()

	if server == nil {
		t.Fatal("server should not be nil")
	}

	// Should have 2 connections
	if mgr.ConnectionCount() != 2 {
		t.Errorf("expected 2 connections, got %d", mgr.ConnectionCount())
	}
}
