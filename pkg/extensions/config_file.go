package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/tools"
)

// ServerConfig is a unified configuration structure for file-based config.
// It combines client, toolkit, and extensions configuration in a single file.
// Suitable for Kubernetes ConfigMaps, Vault, or any file-based config source.
//
// Example YAML:
//
//	trino:
//	  host: trino.example.com
//	  port: 443
//	  user: service_user
//	  password: ${TRINO_PASSWORD}  # Can reference env vars
//	  catalog: hive
//	  schema: default
//	  ssl: true
//	  ssl_verify: true
//	  timeout: 120s
//	  source: my-mcp-server
//
//	toolkit:
//	  default_limit: 1000
//	  max_limit: 10000
//	  default_timeout: 120s
//	  max_timeout: 300s
//
//	extensions:
//	  logging: false
//	  metrics: false
//	  readonly: true
//	  querylog: false
//	  metadata: false
//	  errors: true
type ServerConfig struct {
	// Trino client configuration
	Trino TrinoConfig `json:"trino" yaml:"trino"`

	// Toolkit configuration
	Toolkit ToolkitConfig `json:"toolkit" yaml:"toolkit"`

	// Extensions configuration
	Extensions ExtFileConfig `json:"extensions" yaml:"extensions"`
}

// TrinoConfig maps to client.Config for file-based loading.
type TrinoConfig struct {
	Host      string   `json:"host" yaml:"host"`
	Port      int      `json:"port" yaml:"port"`
	User      string   `json:"user" yaml:"user"`
	Password  string   `json:"password" yaml:"password"`
	Catalog   string   `json:"catalog" yaml:"catalog"`
	Schema    string   `json:"schema" yaml:"schema"`
	SSL       *bool    `json:"ssl" yaml:"ssl"`               // Pointer to distinguish unset from false
	SSLVerify *bool    `json:"ssl_verify" yaml:"ssl_verify"` // Pointer to distinguish unset from false
	Timeout   Duration `json:"timeout" yaml:"timeout"`       // Supports "120s", "2m", etc.
	Source    string   `json:"source" yaml:"source"`
}

// ToolkitConfig maps to tools.Config for file-based loading.
type ToolkitConfig struct {
	DefaultLimit   int               `json:"default_limit" yaml:"default_limit"`
	MaxLimit       int               `json:"max_limit" yaml:"max_limit"`
	DefaultTimeout Duration          `json:"default_timeout" yaml:"default_timeout"`
	MaxTimeout     Duration          `json:"max_timeout" yaml:"max_timeout"`
	Descriptions   map[string]string `json:"descriptions,omitempty" yaml:"descriptions,omitempty"`
}

// ExtFileConfig maps to Config for file-based loading.
type ExtFileConfig struct {
	Logging  *bool `json:"logging" yaml:"logging"`
	Metrics  *bool `json:"metrics" yaml:"metrics"`
	ReadOnly *bool `json:"readonly" yaml:"readonly"`
	QueryLog *bool `json:"querylog" yaml:"querylog"`
	Metadata *bool `json:"metadata" yaml:"metadata"`
	Errors   *bool `json:"errors" yaml:"errors"`
}

// Duration is a wrapper for time.Duration that supports JSON/YAML unmarshaling
// from strings like "120s", "2m", "1h30m".
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		// Try parsing as number (seconds)
		var secs float64
		if err := json.Unmarshal(b, &secs); err != nil {
			return err
		}
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		// Try parsing as number (seconds)
		var secs float64
		if err := value.Decode(&secs); err != nil {
			return err
		}
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Trino: TrinoConfig{
			Host:    "localhost",
			Port:    8080,
			Catalog: "memory",
			Schema:  "default",
			Source:  "mcp-trino",
			Timeout: Duration(120 * time.Second),
		},
		Toolkit: ToolkitConfig{
			DefaultLimit:   1000,
			MaxLimit:       10000,
			DefaultTimeout: Duration(120 * time.Second),
			MaxTimeout:     Duration(300 * time.Second),
		},
		Extensions: ExtFileConfig{},
	}
}

// FromFile loads a ServerConfig from a JSON or YAML file.
// The format is detected by the file extension (.json, .yaml, .yml).
// Missing fields use values from DefaultServerConfig.
func FromFile(path string) (ServerConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- config file path is provided by administrator
	if err != nil {
		return ServerConfig{}, fmt.Errorf("reading config file: %w", err)
	}

	return FromBytes(data, filepath.Ext(path))
}

// FromBytes loads a ServerConfig from bytes.
// The format parameter should be ".json", ".yaml", or ".yml".
// This is useful for loading config from Kubernetes ConfigMaps or other sources.
func FromBytes(data []byte, format string) (ServerConfig, error) {
	cfg := DefaultServerConfig()

	switch format {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return ServerConfig{}, fmt.Errorf("parsing JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return ServerConfig{}, fmt.Errorf("parsing YAML config: %w", err)
		}
	default:
		return ServerConfig{}, fmt.Errorf("unsupported config format: %s (use .json, .yaml, or .yml)", format)
	}

	// Expand environment variables in sensitive fields
	cfg.Trino.Password = os.ExpandEnv(cfg.Trino.Password)
	cfg.Trino.User = os.ExpandEnv(cfg.Trino.User)

	return cfg, nil
}

// ClientConfig converts the Trino section to a client.Config.
func (c ServerConfig) ClientConfig() client.Config {
	cfg := client.DefaultConfig()

	if c.Trino.Host != "" {
		cfg.Host = c.Trino.Host
		// Default to SSL for non-localhost hosts
		if c.Trino.Host != "localhost" && c.Trino.Host != "127.0.0.1" {
			cfg.SSL = true
			cfg.Port = 443
		}
	}
	if c.Trino.Port != 0 {
		cfg.Port = c.Trino.Port
	}
	if c.Trino.User != "" {
		cfg.User = c.Trino.User
	}
	if c.Trino.Password != "" {
		cfg.Password = c.Trino.Password
	}
	if c.Trino.Catalog != "" {
		cfg.Catalog = c.Trino.Catalog
	}
	if c.Trino.Schema != "" {
		cfg.Schema = c.Trino.Schema
	}
	if c.Trino.SSL != nil {
		cfg.SSL = *c.Trino.SSL
	}
	if c.Trino.SSLVerify != nil {
		cfg.SSLVerify = *c.Trino.SSLVerify
	}
	if c.Trino.Timeout.Duration() > 0 {
		cfg.Timeout = c.Trino.Timeout.Duration()
	}
	if c.Trino.Source != "" {
		cfg.Source = c.Trino.Source
	}

	return cfg
}

// ToolsConfig converts the Toolkit section to a tools.Config.
func (c ServerConfig) ToolsConfig() tools.Config {
	cfg := tools.DefaultConfig()

	if c.Toolkit.DefaultLimit > 0 {
		cfg.DefaultLimit = c.Toolkit.DefaultLimit
	}
	if c.Toolkit.MaxLimit > 0 {
		cfg.MaxLimit = c.Toolkit.MaxLimit
	}
	if c.Toolkit.DefaultTimeout.Duration() > 0 {
		cfg.DefaultTimeout = c.Toolkit.DefaultTimeout.Duration()
	}
	if c.Toolkit.MaxTimeout.Duration() > 0 {
		cfg.MaxTimeout = c.Toolkit.MaxTimeout.Duration()
	}

	return cfg
}

// DescriptionsMap converts the Toolkit.Descriptions string map to a tools.ToolName map.
// Returns nil if no descriptions are configured.
func (c ServerConfig) DescriptionsMap() map[tools.ToolName]string {
	if len(c.Toolkit.Descriptions) == 0 {
		return nil
	}
	m := make(map[tools.ToolName]string, len(c.Toolkit.Descriptions))
	for k, v := range c.Toolkit.Descriptions {
		m[tools.ToolName(k)] = v
	}
	return m
}

// ExtConfig converts the Extensions section to a Config.
func (c ServerConfig) ExtConfig() Config {
	cfg := DefaultConfig()

	if c.Extensions.Logging != nil {
		cfg.EnableLogging = *c.Extensions.Logging
	}
	if c.Extensions.Metrics != nil {
		cfg.EnableMetrics = *c.Extensions.Metrics
	}
	if c.Extensions.ReadOnly != nil {
		cfg.EnableReadOnly = *c.Extensions.ReadOnly
	}
	if c.Extensions.QueryLog != nil {
		cfg.EnableQueryLog = *c.Extensions.QueryLog
	}
	if c.Extensions.Metadata != nil {
		cfg.EnableMetadata = *c.Extensions.Metadata
	}
	if c.Extensions.Errors != nil {
		cfg.EnableErrorHelp = *c.Extensions.Errors
	}

	return cfg
}

// LoadConfig loads configuration from multiple sources with precedence:
// 1. File config (if path is provided and file exists)
// 2. Environment variables (override file values)
// 3. Defaults (for any unset values)
//
// This allows flexible deployment patterns:
//   - File-only: Kubernetes ConfigMap with secrets from Vault.
//   - Env-only: Docker/container deployments.
//   - Hybrid: Base config in file, secrets in env vars.
func LoadConfig(path string) (ServerConfig, error) {
	var cfg ServerConfig
	var err error

	// Start with file config or defaults
	if path != "" {
		cfg, err = FromFile(path)
		if err != nil {
			return ServerConfig{}, err
		}
	} else {
		cfg = DefaultServerConfig()
	}

	// Override with environment variables
	cfg = applyEnvOverrides(cfg)

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to a config.
// Environment variables take precedence over file config.
func applyEnvOverrides(cfg ServerConfig) ServerConfig {
	cfg.Trino = applyTrinoEnvOverrides(cfg.Trino)
	cfg.Extensions = applyExtensionsEnvOverrides(cfg.Extensions)
	return cfg
}

// applyTrinoEnvOverrides applies TRINO_* environment variable overrides.
func applyTrinoEnvOverrides(cfg TrinoConfig) TrinoConfig {
	if v := os.Getenv("TRINO_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("TRINO_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("TRINO_USER"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("TRINO_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("TRINO_CATALOG"); v != "" {
		cfg.Catalog = v
	}
	if v := os.Getenv("TRINO_SCHEMA"); v != "" {
		cfg.Schema = v
	}
	if v := os.Getenv("TRINO_SSL"); v != "" {
		b := parseBool(v)
		cfg.SSL = &b
	}
	if v := os.Getenv("TRINO_SSL_VERIFY"); v != "" {
		b := parseBool(v)
		cfg.SSLVerify = &b
	}
	if v := os.Getenv("TRINO_SOURCE"); v != "" {
		cfg.Source = v
	}
	return cfg
}

// applyExtensionsEnvOverrides applies MCP_TRINO_EXT_* environment variable overrides.
func applyExtensionsEnvOverrides(cfg ExtFileConfig) ExtFileConfig {
	if v := os.Getenv("MCP_TRINO_EXT_LOGGING"); v != "" {
		b := parseBool(v)
		cfg.Logging = &b
	}
	if v := os.Getenv("MCP_TRINO_EXT_METRICS"); v != "" {
		b := parseBool(v)
		cfg.Metrics = &b
	}
	if v := os.Getenv("MCP_TRINO_EXT_READONLY"); v != "" {
		b := parseBool(v)
		cfg.ReadOnly = &b
	}
	if v := os.Getenv("MCP_TRINO_EXT_QUERYLOG"); v != "" {
		b := parseBool(v)
		cfg.QueryLog = &b
	}
	if v := os.Getenv("MCP_TRINO_EXT_METADATA"); v != "" {
		b := parseBool(v)
		cfg.Metadata = &b
	}
	if v := os.Getenv("MCP_TRINO_EXT_ERRORS"); v != "" {
		b := parseBool(v)
		cfg.Errors = &b
	}
	return cfg
}
