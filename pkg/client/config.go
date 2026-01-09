// Package client provides a Trino client wrapper for MCP servers.
package client

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for connecting to a Trino server.
type Config struct {
	// Host is the Trino server hostname (without protocol or port).
	Host string

	// Port is the Trino server port. Default: 443 for SSL, 8080 otherwise.
	Port int

	// User is the Trino username for authentication.
	User string

	// Password is the Trino password (optional, for password auth).
	Password string

	// Catalog is the default catalog to use.
	Catalog string

	// Schema is the default schema to use.
	Schema string

	// SSL enables HTTPS connection. Default: true.
	SSL bool

	// SSLVerify enables SSL certificate verification. Default: true.
	SSLVerify bool

	// Timeout is the default query timeout. Default: 120s.
	Timeout time.Duration

	// Source identifies this client to Trino. Default: "mcp-trino".
	Source string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Host:      "localhost",
		Port:      8080,
		SSL:       false,
		SSLVerify: true,
		Timeout:   120 * time.Second,
		Source:    "mcp-trino",
		Catalog:   "memory",
		Schema:    "default",
	}
}

// FromEnv loads configuration from environment variables.
// Environment variables:
//   - TRINO_HOST: Trino server hostname
//   - TRINO_PORT: Trino server port
//   - TRINO_USER: Trino username
//   - TRINO_PASSWORD: Trino password
//   - TRINO_CATALOG: Default catalog
//   - TRINO_SCHEMA: Default schema
//   - TRINO_SSL: Enable SSL (true/false)
//   - TRINO_SSL_VERIFY: Verify SSL certificates (true/false)
//   - TRINO_TIMEOUT: Query timeout in seconds
//   - TRINO_SOURCE: Client source identifier
func FromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("TRINO_HOST"); v != "" {
		cfg.Host = v
		// Default to SSL for non-localhost hosts
		if v != "localhost" && v != "127.0.0.1" {
			cfg.SSL = true
			cfg.Port = 443
		}
	}

	if v := os.Getenv("TRINO_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
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
		cfg.SSL = v == "true" || v == "1"
	}

	if v := os.Getenv("TRINO_SSL_VERIFY"); v != "" {
		cfg.SSLVerify = v == "true" || v == "1"
	}

	if v := os.Getenv("TRINO_TIMEOUT"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			cfg.Timeout = time.Duration(secs) * time.Second
		}
	}

	if v := os.Getenv("TRINO_SOURCE"); v != "" {
		cfg.Source = v
	}

	return cfg
}

// DSN returns the data source name for the Trino driver.
func (c Config) DSN() string {
	scheme := "http"
	if c.SSL {
		scheme = "https"
	}

	// Build basic DSN
	dsn := fmt.Sprintf("%s://%s@%s:%d", scheme, c.User, c.Host, c.Port)

	// Add catalog and schema if set
	if c.Catalog != "" {
		dsn += "/" + c.Catalog
		if c.Schema != "" {
			dsn += "/" + c.Schema
		}
	}

	// Add query parameters
	params := fmt.Sprintf("?source=%s", c.Source)
	if !c.SSLVerify && c.SSL {
		params += "&sslVerify=false"
	}

	return dsn + params
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}
