package extensions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFromBytes_JSON(t *testing.T) {
	jsonConfig := `{
		"trino": {
			"host": "trino.example.com",
			"port": 443,
			"user": "test_user",
			"password": "test_pass",
			"catalog": "hive",
			"schema": "analytics",
			"ssl": true,
			"ssl_verify": true,
			"timeout": "60s",
			"source": "test-server"
		},
		"toolkit": {
			"default_limit": 500,
			"max_limit": 5000,
			"default_timeout": "60s",
			"max_timeout": "180s"
		},
		"extensions": {
			"logging": true,
			"metrics": true,
			"readonly": true,
			"querylog": false,
			"metadata": true,
			"errors": true
		}
	}`

	cfg, err := FromBytes([]byte(jsonConfig), ".json")
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	// Check Trino config
	if cfg.Trino.Host != "trino.example.com" {
		t.Errorf("expected host 'trino.example.com', got %q", cfg.Trino.Host)
	}
	if cfg.Trino.Port != 443 {
		t.Errorf("expected port 443, got %d", cfg.Trino.Port)
	}
	if cfg.Trino.User != "test_user" {
		t.Errorf("expected user 'test_user', got %q", cfg.Trino.User)
	}
	if cfg.Trino.Timeout.Duration() != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", cfg.Trino.Timeout.Duration())
	}

	// Check toolkit config
	if cfg.Toolkit.DefaultLimit != 500 {
		t.Errorf("expected default_limit 500, got %d", cfg.Toolkit.DefaultLimit)
	}
	if cfg.Toolkit.MaxLimit != 5000 {
		t.Errorf("expected max_limit 5000, got %d", cfg.Toolkit.MaxLimit)
	}

	// Check extensions config
	if cfg.Extensions.Logging == nil || !*cfg.Extensions.Logging {
		t.Error("expected logging to be true")
	}
	if cfg.Extensions.Metrics == nil || !*cfg.Extensions.Metrics {
		t.Error("expected metrics to be true")
	}
}

func TestFromBytes_YAML(t *testing.T) {
	yamlConfig := `
trino:
  host: trino.staging.example.com
  port: 8443
  user: staging_user
  catalog: iceberg
  schema: staging
  ssl: true
  timeout: 90s

toolkit:
  default_limit: 2000
  max_limit: 20000

extensions:
  logging: true
  readonly: true
`

	cfg, err := FromBytes([]byte(yamlConfig), ".yaml")
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	if cfg.Trino.Host != "trino.staging.example.com" {
		t.Errorf("expected host 'trino.staging.example.com', got %q", cfg.Trino.Host)
	}
	if cfg.Trino.Port != 8443 {
		t.Errorf("expected port 8443, got %d", cfg.Trino.Port)
	}
	if cfg.Trino.Timeout.Duration() != 90*time.Second {
		t.Errorf("expected timeout 90s, got %v", cfg.Trino.Timeout.Duration())
	}
	if cfg.Toolkit.DefaultLimit != 2000 {
		t.Errorf("expected default_limit 2000, got %d", cfg.Toolkit.DefaultLimit)
	}
}

func TestFromBytes_DurationAsNumber(t *testing.T) {
	jsonConfig := `{
		"trino": {
			"timeout": 120
		},
		"toolkit": {
			"default_timeout": 60
		}
	}`

	cfg, err := FromBytes([]byte(jsonConfig), ".json")
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	if cfg.Trino.Timeout.Duration() != 120*time.Second {
		t.Errorf("expected timeout 120s, got %v", cfg.Trino.Timeout.Duration())
	}
	if cfg.Toolkit.DefaultTimeout.Duration() != 60*time.Second {
		t.Errorf("expected default_timeout 60s, got %v", cfg.Toolkit.DefaultTimeout.Duration())
	}
}

func TestFromBytes_UnsupportedFormat(t *testing.T) {
	_, err := FromBytes([]byte("{}"), ".toml")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestFromFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlConfig := `
trino:
  host: file-test.example.com
  user: file_user
`

	if err := os.WriteFile(configPath, []byte(yamlConfig), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	cfg, err := FromFile(configPath)
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	if cfg.Trino.Host != "file-test.example.com" {
		t.Errorf("expected host 'file-test.example.com', got %q", cfg.Trino.Host)
	}
}

func TestFromFile_NotFound(t *testing.T) {
	_, err := FromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestServerConfig_ClientConfig(t *testing.T) {
	ssl := true
	sslVerify := false
	cfg := ServerConfig{
		Trino: TrinoConfig{
			Host:      "convert.example.com",
			Port:      8080,
			User:      "convert_user",
			Password:  "convert_pass",
			Catalog:   "delta",
			Schema:    "bronze",
			SSL:       &ssl,
			SSLVerify: &sslVerify,
			Timeout:   Duration(90 * time.Second),
			Source:    "test-convert",
		},
	}

	clientCfg := cfg.ClientConfig()

	if clientCfg.Host != "convert.example.com" {
		t.Errorf("expected host 'convert.example.com', got %q", clientCfg.Host)
	}
	if clientCfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", clientCfg.Port)
	}
	if clientCfg.User != "convert_user" {
		t.Errorf("expected user 'convert_user', got %q", clientCfg.User)
	}
	if clientCfg.Password != "convert_pass" {
		t.Errorf("expected password 'convert_pass', got %q", clientCfg.Password)
	}
	if !clientCfg.SSL {
		t.Error("expected SSL to be true")
	}
	if clientCfg.SSLVerify {
		t.Error("expected SSLVerify to be false")
	}
	if clientCfg.Timeout != 90*time.Second {
		t.Errorf("expected timeout 90s, got %v", clientCfg.Timeout)
	}
}

func TestServerConfig_ToolsConfig(t *testing.T) {
	cfg := ServerConfig{
		Toolkit: ToolkitConfig{
			DefaultLimit:   500,
			MaxLimit:       5000,
			DefaultTimeout: Duration(30 * time.Second),
			MaxTimeout:     Duration(120 * time.Second),
		},
	}

	toolsCfg := cfg.ToolsConfig()

	if toolsCfg.DefaultLimit != 500 {
		t.Errorf("expected DefaultLimit 500, got %d", toolsCfg.DefaultLimit)
	}
	if toolsCfg.MaxLimit != 5000 {
		t.Errorf("expected MaxLimit 5000, got %d", toolsCfg.MaxLimit)
	}
	if toolsCfg.DefaultTimeout != 30*time.Second {
		t.Errorf("expected DefaultTimeout 30s, got %v", toolsCfg.DefaultTimeout)
	}
	if toolsCfg.MaxTimeout != 120*time.Second {
		t.Errorf("expected MaxTimeout 120s, got %v", toolsCfg.MaxTimeout)
	}
}

func TestServerConfig_ExtConfig(t *testing.T) {
	logging := true
	metrics := true
	readonly := false
	querylog := true
	metadata := true
	errors := false

	cfg := ServerConfig{
		Extensions: ExtFileConfig{
			Logging:  &logging,
			Metrics:  &metrics,
			ReadOnly: &readonly,
			QueryLog: &querylog,
			Metadata: &metadata,
			Errors:   &errors,
		},
	}

	extCfg := cfg.ExtConfig()

	if !extCfg.EnableLogging {
		t.Error("expected EnableLogging to be true")
	}
	if !extCfg.EnableMetrics {
		t.Error("expected EnableMetrics to be true")
	}
	if extCfg.EnableReadOnly {
		t.Error("expected EnableReadOnly to be false")
	}
	if !extCfg.EnableQueryLog {
		t.Error("expected EnableQueryLog to be true")
	}
	if !extCfg.EnableMetadata {
		t.Error("expected EnableMetadata to be true")
	}
	if extCfg.EnableErrorHelp {
		t.Error("expected EnableErrorHelp to be false")
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlConfig := `
trino:
  host: file-host.example.com
  user: file_user
extensions:
  logging: false
`

	if err := os.WriteFile(configPath, []byte(yamlConfig), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Set env overrides
	t.Setenv("TRINO_HOST", "env-host.example.com")
	t.Setenv("MCP_TRINO_EXT_LOGGING", "true")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Env should override file
	if cfg.Trino.Host != "env-host.example.com" {
		t.Errorf("expected host 'env-host.example.com', got %q", cfg.Trino.Host)
	}

	// Non-overridden values should come from file
	if cfg.Trino.User != "file_user" {
		t.Errorf("expected user 'file_user', got %q", cfg.Trino.User)
	}

	// Extension env override
	if cfg.Extensions.Logging == nil || !*cfg.Extensions.Logging {
		t.Error("expected logging to be true from env override")
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Clear env vars
	envVars := []string{
		"TRINO_HOST", "TRINO_PORT", "TRINO_USER", "TRINO_PASSWORD",
		"MCP_TRINO_EXT_LOGGING", "MCP_TRINO_EXT_READONLY",
	}
	for _, v := range envVars {
		t.Setenv(v, "")
	}

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should have defaults
	if cfg.Trino.Host != "localhost" {
		t.Errorf("expected default host 'localhost', got %q", cfg.Trino.Host)
	}
	if cfg.Toolkit.DefaultLimit != 1000 {
		t.Errorf("expected default limit 1000, got %d", cfg.Toolkit.DefaultLimit)
	}
}

func TestEnvironmentVariableExpansion(t *testing.T) {
	t.Setenv("TEST_TRINO_PASSWORD", "secret123")

	jsonConfig := `{
		"trino": {
			"user": "admin",
			"password": "${TEST_TRINO_PASSWORD}"
		}
	}`

	cfg, err := FromBytes([]byte(jsonConfig), ".json")
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	if cfg.Trino.Password != "secret123" {
		t.Errorf("expected password 'secret123', got %q", cfg.Trino.Password)
	}
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	// Verify defaults
	if cfg.Trino.Host != "localhost" {
		t.Errorf("expected default host 'localhost', got %q", cfg.Trino.Host)
	}
	if cfg.Trino.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Trino.Port)
	}
	if cfg.Trino.Catalog != "memory" {
		t.Errorf("expected default catalog 'memory', got %q", cfg.Trino.Catalog)
	}
	if cfg.Toolkit.DefaultLimit != 1000 {
		t.Errorf("expected default limit 1000, got %d", cfg.Toolkit.DefaultLimit)
	}
	if cfg.Toolkit.MaxLimit != 10000 {
		t.Errorf("expected max limit 10000, got %d", cfg.Toolkit.MaxLimit)
	}
}

func TestServerConfig_ClientConfig_SSLDefault(t *testing.T) {
	// When host is non-localhost and SSL not specified, should default to true
	cfg := ServerConfig{
		Trino: TrinoConfig{
			Host: "trino.production.example.com",
			User: "prod_user",
		},
	}

	clientCfg := cfg.ClientConfig()

	if !clientCfg.SSL {
		t.Error("expected SSL to default to true for non-localhost host")
	}
	if clientCfg.Port != 443 {
		t.Errorf("expected port 443 for SSL, got %d", clientCfg.Port)
	}
}

func TestFromBytes_PartialConfig(t *testing.T) {
	// Test that partial config works and defaults are preserved
	yamlConfig := `
trino:
  host: partial.example.com
`

	cfg, err := FromBytes([]byte(yamlConfig), ".yaml")
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	// Specified value
	if cfg.Trino.Host != "partial.example.com" {
		t.Errorf("expected host 'partial.example.com', got %q", cfg.Trino.Host)
	}

	// Defaults should be preserved
	if cfg.Toolkit.DefaultLimit != 1000 {
		t.Errorf("expected default limit 1000, got %d", cfg.Toolkit.DefaultLimit)
	}
	if cfg.Trino.Timeout.Duration() != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", cfg.Trino.Timeout.Duration())
	}
}
