package server

import (
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
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

	// Verify ClientConfig has defaults from environment
	if opts.ClientConfig.Host == "" {
		t.Error("ClientConfig.Host should have default value")
	}

	// Verify ToolkitConfig has defaults
	if opts.ToolkitConfig.DefaultLimit <= 0 {
		t.Error("ToolkitConfig.DefaultLimit should be positive")
	}
	if opts.ToolkitConfig.MaxLimit <= 0 {
		t.Error("ToolkitConfig.MaxLimit should be positive")
	}
}

func TestOptions_CustomClientConfig(t *testing.T) {
	opts := Options{
		ClientConfig: client.Config{
			Host:    "custom-host",
			Port:    9999,
			User:    "testuser",
			Catalog: "testcatalog",
			Schema:  "testschema",
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	if opts.ClientConfig.Host != "custom-host" {
		t.Errorf("expected Host 'custom-host', got %q", opts.ClientConfig.Host)
	}
	if opts.ClientConfig.Port != 9999 {
		t.Errorf("expected Port 9999, got %d", opts.ClientConfig.Port)
	}
}

func TestOptions_CustomToolkitConfig(t *testing.T) {
	opts := Options{
		ClientConfig: client.DefaultConfig(),
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
	opts := Options{
		ClientConfig: client.Config{
			Host: "", // Invalid: empty host
			User: "admin",
			Port: 8080,
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, trinoClient, err := New(opts)
	if err == nil {
		t.Error("expected error for invalid config")
		if server != nil || trinoClient != nil {
			t.Error("server and client should be nil on error")
		}
	}
}

func TestNew_MissingUser(t *testing.T) {
	opts := Options{
		ClientConfig: client.Config{
			Host: "localhost",
			User: "", // Invalid: missing user
			Port: 8080,
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, trinoClient, err := New(opts)
	if err == nil {
		t.Error("expected error for missing user")
		if trinoClient != nil {
			trinoClient.Close()
		}
	}
	_ = server
}

func TestNew_InvalidPort(t *testing.T) {
	opts := Options{
		ClientConfig: client.Config{
			Host: "localhost",
			User: "admin",
			Port: 0, // Invalid: port 0
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, trinoClient, err := New(opts)
	if err == nil {
		t.Error("expected error for invalid port")
		if trinoClient != nil {
			trinoClient.Close()
		}
	}
	_ = server
}

func TestNew_ValidConfig(t *testing.T) {
	opts := Options{
		ClientConfig: client.Config{
			Host:    "localhost",
			Port:    8080,
			User:    "admin",
			SSL:     false,
			Source:  "test",
			Catalog: "memory",
			Schema:  "default",
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, trinoClient, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if server == nil {
		t.Error("server should not be nil")
	}
	if trinoClient == nil {
		t.Error("trinoClient should not be nil")
	}

	// Clean up
	if trinoClient != nil {
		trinoClient.Close()
	}
}

func TestNew_ServerImplementation(t *testing.T) {
	opts := Options{
		ClientConfig: client.Config{
			Host:    "localhost",
			Port:    8080,
			User:    "admin",
			SSL:     false,
			Source:  "test",
			Catalog: "memory",
			Schema:  "default",
		},
		ToolkitConfig: tools.DefaultConfig(),
	}

	server, trinoClient, err := New(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer trinoClient.Close()

	// Server should be configured with proper implementation name
	if server == nil {
		t.Fatal("server should not be nil")
	}
}
