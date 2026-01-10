package multiserver

import (
	"testing"

	"github.com/txn2/mcp-trino/pkg/client"
)

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantErr     bool
		wantCount   int
		wantDefault string
	}{
		{
			name: "primary only",
			envVars: map[string]string{
				"TRINO_HOST": "localhost",
				"TRINO_USER": "admin",
			},
			wantErr:     false,
			wantCount:   1,
			wantDefault: "Database",
		},
		{
			name: "custom connection name",
			envVars: map[string]string{
				"TRINO_HOST":            "localhost",
				"TRINO_USER":            "admin",
				"TRINO_CONNECTION_NAME": "Production Data",
			},
			wantErr:     false,
			wantCount:   1,
			wantDefault: "Production Data",
		},
		{
			name: "with additional servers",
			envVars: map[string]string{
				"TRINO_HOST":               "prod.example.com",
				"TRINO_USER":               "admin",
				"TRINO_ADDITIONAL_SERVERS": `{"staging": {"host": "staging.example.com"}}`,
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "multiple additional servers",
			envVars: map[string]string{
				"TRINO_HOST":               "prod.example.com",
				"TRINO_USER":               "admin",
				"TRINO_ADDITIONAL_SERVERS": `{"staging": {"host": "staging.example.com"}, "dev": {"host": "localhost", "port": 8080}}`,
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "invalid JSON",
			envVars: map[string]string{
				"TRINO_HOST":               "localhost",
				"TRINO_USER":               "admin",
				"TRINO_ADDITIONAL_SERVERS": `{invalid json}`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set env vars
			for k := range tt.envVars {
				t.Setenv(k, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := FromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("FromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got := cfg.ConnectionCount(); got != tt.wantCount {
				t.Errorf("ConnectionCount() = %d, want %d", got, tt.wantCount)
			}
			if tt.wantDefault != "" && cfg.Default != tt.wantDefault {
				t.Errorf("Default = %q, want %q", cfg.Default, tt.wantDefault)
			}
		})
	}
}

func TestConfig_ClientConfig(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host:     "prod.example.com",
			Port:     443,
			User:     "admin",
			Password: "secret",
			Catalog:  "hive",
			Schema:   "default",
			SSL:      true,
		},
		Connections: map[string]ConnectionConfig{
			"staging": {
				Host:    "staging.example.com",
				Catalog: "iceberg",
			},
			"dev": {
				Host: "localhost",
				Port: 8080,
				SSL:  boolPtr(false),
			},
		},
	}

	tests := []struct {
		name     string
		connName string
		wantHost string
		wantPort int
		wantUser string
		wantCat  string
		wantSSL  bool
		wantErr  bool
	}{
		{
			name:     "default connection",
			connName: "default",
			wantHost: "prod.example.com",
			wantPort: 443,
			wantUser: "admin",
			wantCat:  "hive",
			wantSSL:  true,
		},
		{
			name:     "empty name returns default",
			connName: "",
			wantHost: "prod.example.com",
			wantPort: 443,
			wantUser: "admin",
			wantCat:  "hive",
			wantSSL:  true,
		},
		{
			name:     "staging inherits user/password",
			connName: "staging",
			wantHost: "staging.example.com",
			wantPort: 443, // SSL default
			wantUser: "admin",
			wantCat:  "iceberg", // overridden
			wantSSL:  true,
		},
		{
			name:     "dev with explicit non-SSL",
			connName: "dev",
			wantHost: "localhost",
			wantPort: 8080,
			wantUser: "admin",
			wantCat:  "hive", // inherited
			wantSSL:  false,
		},
		{
			name:     "unknown connection",
			connName: "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.ClientConfig(tt.connName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientConfig(%q) error = %v, wantErr %v", tt.connName, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", got.Host, tt.wantHost)
			}
			if got.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", got.Port, tt.wantPort)
			}
			if got.User != tt.wantUser {
				t.Errorf("User = %q, want %q", got.User, tt.wantUser)
			}
			if got.Catalog != tt.wantCat {
				t.Errorf("Catalog = %q, want %q", got.Catalog, tt.wantCat)
			}
			if got.SSL != tt.wantSSL {
				t.Errorf("SSL = %v, want %v", got.SSL, tt.wantSSL)
			}
		})
	}
}

func TestConfig_ConnectionNames(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{Host: "localhost"},
		Connections: map[string]ConnectionConfig{
			"staging": {Host: "staging.example.com"},
			"dev":     {Host: "localhost"},
		},
	}

	names := cfg.ConnectionNames()
	if len(names) != 3 {
		t.Errorf("ConnectionNames() returned %d names, want 3", len(names))
	}
	if names[0] != "default" {
		t.Errorf("First name = %q, want %q", names[0], "default")
	}
}

func TestConfig_ConnectionInfos(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host:    "prod.example.com",
			Port:    443,
			Catalog: "hive",
			Schema:  "default",
			SSL:     true,
		},
		Connections: map[string]ConnectionConfig{
			"staging": {Host: "staging.example.com", Catalog: "iceberg"},
		},
	}

	infos := cfg.ConnectionInfos()
	if len(infos) != 2 {
		t.Errorf("ConnectionInfos() returned %d infos, want 2", len(infos))
	}

	// Check default connection
	var defaultInfo *ConnectionInfo
	for i := range infos {
		if infos[i].Name == "default" {
			defaultInfo = &infos[i]
			break
		}
	}
	if defaultInfo == nil {
		t.Fatal("default connection not found in infos")
	}
	if !defaultInfo.IsDefault {
		t.Error("default connection should have IsDefault = true")
	}
	if defaultInfo.Host != "prod.example.com" {
		t.Errorf("default Host = %q, want %q", defaultInfo.Host, "prod.example.com")
	}
}

func TestManager_HasConnection(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{Host: "localhost", User: "admin"},
		Connections: map[string]ConnectionConfig{
			"staging": {Host: "staging.example.com"},
		},
	}
	mgr := NewManager(cfg)

	tests := []struct {
		name string
		want bool
	}{
		{"", true}, // empty = default
		{"default", true},
		{"staging", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mgr.HasConnection(tt.name); got != tt.want {
				t.Errorf("HasConnection(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestManager_Connections(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{Host: "localhost", User: "admin"},
		Connections: map[string]ConnectionConfig{
			"staging": {Host: "staging.example.com"},
			"dev":     {Host: "localhost"},
		},
	}
	mgr := NewManager(cfg)

	conns := mgr.Connections()
	if len(conns) != 3 {
		t.Errorf("Connections() returned %d, want 3", len(conns))
	}
}

func TestManager_ConnectionCount(t *testing.T) {
	cfg := Config{
		Default:     "default",
		Primary:     client.Config{Host: "localhost", User: "admin"},
		Connections: map[string]ConnectionConfig{},
	}
	mgr := NewManager(cfg)

	if got := mgr.ConnectionCount(); got != 1 {
		t.Errorf("ConnectionCount() = %d, want 1", got)
	}

	cfg.Connections["staging"] = ConnectionConfig{Host: "staging.example.com"}
	mgr = NewManager(cfg)
	if got := mgr.ConnectionCount(); got != 2 {
		t.Errorf("ConnectionCount() = %d, want 2", got)
	}
}

func TestManager_Client_UnknownConnection(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{Host: "localhost", User: "admin"},
	}
	mgr := NewManager(cfg)

	_, err := mgr.Client("unknown")
	if err == nil {
		t.Error("Client(\"unknown\") should return error")
	}
}

func TestSingleClientManager(t *testing.T) {
	// Create a mock config (we won't actually connect)
	cfg := client.Config{
		Host:    "test.example.com",
		Port:    443,
		User:    "test",
		Catalog: "hive",
	}

	// SingleClientManager should work without an actual client for the wrapper
	mgr := &Manager{
		config: Config{
			Default:     "default",
			Primary:     cfg,
			Connections: make(map[string]ConnectionConfig),
		},
		clients: make(map[string]*client.Client),
	}

	if got := mgr.ConnectionCount(); got != 1 {
		t.Errorf("ConnectionCount() = %d, want 1", got)
	}

	if !mgr.HasConnection("default") {
		t.Error("HasConnection(\"default\") = false, want true")
	}

	if mgr.HasConnection("staging") {
		t.Error("HasConnection(\"staging\") = true, want false")
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestNewManagerFromEnv(t *testing.T) {
	// Set up environment for valid config
	t.Setenv("TRINO_HOST", "localhost")
	t.Setenv("TRINO_PORT", "8080")
	t.Setenv("TRINO_USER", "admin")
	t.Setenv("TRINO_SSL", "false")
	t.Setenv("TRINO_ADDITIONAL_SERVERS", "")

	mgr, err := NewManagerFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.ConnectionCount() != 1 {
		t.Errorf("expected 1 connection, got %d", mgr.ConnectionCount())
	}
}

func TestNewManagerFromEnv_WithAdditionalServers(t *testing.T) {
	t.Setenv("TRINO_HOST", "prod.example.com")
	t.Setenv("TRINO_USER", "admin")
	t.Setenv("TRINO_ADDITIONAL_SERVERS", `{"staging": {"host": "staging.example.com"}}`)

	mgr, err := NewManagerFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr.ConnectionCount() != 2 {
		t.Errorf("expected 2 connections, got %d", mgr.ConnectionCount())
	}
}

func TestNewManagerFromEnv_InvalidJSON(t *testing.T) {
	t.Setenv("TRINO_HOST", "localhost")
	t.Setenv("TRINO_USER", "admin")
	t.Setenv("TRINO_ADDITIONAL_SERVERS", `{invalid}`)

	_, err := NewManagerFromEnv()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestManager_DefaultClient(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host: "localhost",
			Port: 8080,
			User: "admin",
			SSL:  false,
		},
	}
	mgr := NewManager(cfg)

	// DefaultClient should return the same as Client("default")
	c1, err := mgr.DefaultClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c2, err := mgr.Client("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c1 != c2 {
		t.Error("DefaultClient should return same client as Client(\"default\")")
	}

	// Clean up
	mgr.Close()
}

func TestManager_ConnectionInfos(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host:    "prod.example.com",
			Port:    443,
			User:    "admin",
			Catalog: "hive",
			Schema:  "default",
			SSL:     true,
		},
		Connections: map[string]ConnectionConfig{
			"staging": {Host: "staging.example.com", Catalog: "iceberg"},
		},
	}
	mgr := NewManager(cfg)

	infos := mgr.ConnectionInfos()
	if len(infos) != 2 {
		t.Errorf("expected 2 connection infos, got %d", len(infos))
	}

	// Verify default connection info
	var defaultInfo *ConnectionInfo
	for i := range infos {
		if infos[i].Name == "default" {
			defaultInfo = &infos[i]
			break
		}
	}
	if defaultInfo == nil {
		t.Fatal("default connection not found")
	}
	if !defaultInfo.IsDefault {
		t.Error("default connection should have IsDefault=true")
	}
	if defaultInfo.Host != "prod.example.com" {
		t.Errorf("expected host 'prod.example.com', got %q", defaultInfo.Host)
	}
}

func TestManager_Config(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host: "localhost",
			Port: 8080,
			User: "admin",
		},
		Connections: map[string]ConnectionConfig{
			"staging": {Host: "staging.example.com"},
		},
	}
	mgr := NewManager(cfg)

	returnedCfg := mgr.Config()
	if returnedCfg.Default != cfg.Default {
		t.Errorf("expected Default %q, got %q", cfg.Default, returnedCfg.Default)
	}
	if returnedCfg.Primary.Host != cfg.Primary.Host {
		t.Errorf("expected Primary.Host %q, got %q", cfg.Primary.Host, returnedCfg.Primary.Host)
	}
	if len(returnedCfg.Connections) != len(cfg.Connections) {
		t.Errorf("expected %d connections, got %d", len(cfg.Connections), len(returnedCfg.Connections))
	}
}

func TestManager_Close(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host: "localhost",
			Port: 8080,
			User: "admin",
			SSL:  false,
		},
	}
	mgr := NewManager(cfg)

	// Create a client
	_, err := mgr.Client("default")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	// Close should work
	err = mgr.Close()
	if err != nil {
		t.Errorf("unexpected error closing: %v", err)
	}

	// After close, client cache should be empty (new client created on next access)
	// This is hard to test directly, but we can verify Close doesn't error on empty cache
	err = mgr.Close()
	if err != nil {
		t.Errorf("unexpected error closing empty manager: %v", err)
	}
}

func TestManager_Close_NoClients(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host: "localhost",
			Port: 8080,
			User: "admin",
		},
	}
	mgr := NewManager(cfg)

	// Close without any clients created
	err := mgr.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSingleClientManager_Full(t *testing.T) {
	// Create a real client config (we'll just test the manager structure)
	cfg := client.Config{
		Host:    "test.example.com",
		Port:    443,
		User:    "test",
		Catalog: "hive",
		Schema:  "default",
		SSL:     true,
	}

	// Use nil client to test manager structure without actual connection
	mgr := SingleClientManager(nil, cfg)

	if mgr.ConnectionCount() != 1 {
		t.Errorf("expected 1 connection, got %d", mgr.ConnectionCount())
	}

	if !mgr.HasConnection("Database") {
		t.Error("expected HasConnection(\"Database\") to return true")
	}

	if mgr.HasConnection("staging") {
		t.Error("expected HasConnection(\"staging\") to return false")
	}

	conns := mgr.Connections()
	if len(conns) != 1 {
		t.Errorf("expected 1 connection name, got %d", len(conns))
	}
	if conns[0] != "Database" {
		t.Errorf("expected connection name 'Database', got %q", conns[0])
	}

	infos := mgr.ConnectionInfos()
	if len(infos) != 1 {
		t.Errorf("expected 1 connection info, got %d", len(infos))
	}

	// Config should be accessible
	returnedCfg := mgr.Config()
	if returnedCfg.Primary.Host != cfg.Host {
		t.Errorf("expected host %q, got %q", cfg.Host, returnedCfg.Primary.Host)
	}
}

func TestManager_Client_ConcurrentAccess(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host: "localhost",
			Port: 8080,
			User: "admin",
			SSL:  false,
		},
	}
	mgr := NewManager(cfg)
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close manager: %v", err)
		}
	}()

	// Concurrent access should be safe
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			// Intentionally ignore error - testing concurrent access safety
			if _, err := mgr.Client("default"); err != nil {
				// Error is expected in some cases, just testing for races
				_ = err
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestManager_Client_InvalidConfig(t *testing.T) {
	cfg := Config{
		Default: "default",
		Primary: client.Config{
			Host: "localhost",
			Port: 8080,
			User: "admin",
		},
		Connections: map[string]ConnectionConfig{
			"invalid": {
				Host: "test.example.com",
				Port: 99999, // Invalid: port > 65535
			},
		},
	}
	mgr := NewManager(cfg)

	_, err := mgr.Client("invalid")
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
