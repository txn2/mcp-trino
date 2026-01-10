// Package multiserver provides support for managing connections to multiple Trino servers.
package multiserver

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/txn2/mcp-trino/pkg/client"
)

// DefaultConnectionName is the fallback name for the primary connection.
const DefaultConnectionName = "database"

// ConnectionConfig defines configuration for a single Trino connection.
// Fields that are empty/zero inherit from the primary connection.
type ConnectionConfig struct {
	// Host is the Trino server hostname (required for additional servers).
	Host string `json:"host"`

	// Port is the Trino server port. Defaults to 443 (SSL) or 8080 (non-SSL).
	Port int `json:"port,omitempty"`

	// User is the Trino username. Inherits from primary if empty.
	User string `json:"user,omitempty"`

	// Password is the Trino password. Inherits from primary if empty.
	Password string `json:"password,omitempty"`

	// Catalog is the default catalog. Inherits from primary if empty.
	Catalog string `json:"catalog,omitempty"`

	// Schema is the default schema. Inherits from primary if empty.
	Schema string `json:"schema,omitempty"`

	// SSL enables HTTPS. Nil means auto-detect based on host.
	SSL *bool `json:"ssl,omitempty"`

	// SSLVerify enables SSL certificate verification. Inherits from primary if nil.
	SSLVerify *bool `json:"ssl_verify,omitempty"`
}

// Config holds configuration for multiple Trino connections.
type Config struct {
	// Default is the display name of the primary connection.
	// This can be customized via TRINO_CONNECTION_NAME env var.
	// Defaults to "Database" if not specified.
	Default string

	// Primary is the primary connection configuration (from TRINO_* env vars).
	Primary client.Config

	// Connections maps connection names to their configurations.
	// The primary connection is always available under the Default name.
	Connections map[string]ConnectionConfig
}

// FromEnv builds a multi-server configuration from environment variables.
//
// Primary server configuration comes from standard TRINO_* variables.
// Additional servers come from TRINO_ADDITIONAL_SERVERS as JSON:
//
//	{"staging": {"host": "staging.example.com", "catalog": "iceberg"}}
//
// The primary connection name can be customized via TRINO_CONNECTION_NAME
// (defaults to "Database").
//
// Additional servers inherit user, password, catalog, schema from the primary
// if not explicitly specified.
func FromEnv() (Config, error) {
	primary := client.FromEnv()

	// Get connection name from env or use default
	connectionName := os.Getenv("TRINO_CONNECTION_NAME")
	if connectionName == "" {
		connectionName = DefaultConnectionName
	}

	cfg := Config{
		Default:     connectionName,
		Primary:     primary,
		Connections: make(map[string]ConnectionConfig),
	}

	// Parse additional servers from JSON env var
	additionalJSON := os.Getenv("TRINO_ADDITIONAL_SERVERS")
	if additionalJSON != "" {
		var additional map[string]ConnectionConfig
		if err := json.Unmarshal([]byte(additionalJSON), &additional); err != nil {
			return Config{}, fmt.Errorf("parsing TRINO_ADDITIONAL_SERVERS: %w", err)
		}
		cfg.Connections = additional
	}

	return cfg, nil
}

// ClientConfig returns a client.Config for the named connection.
// Returns the primary config if name is empty or matches the default connection name.
// Returns an error if the connection name is not found.
func (c Config) ClientConfig(name string) (client.Config, error) {
	// Empty or default connection name returns primary
	if name == "" || name == c.Default {
		return c.Primary, nil
	}

	// Look up additional connection
	conn, ok := c.Connections[name]
	if !ok {
		return client.Config{}, fmt.Errorf("unknown connection: %q (available: %v)", name, c.ConnectionNames())
	}

	// Build config by inheriting from primary
	cfg := c.Primary // Start with primary values

	// Override with connection-specific values
	if conn.Host != "" {
		cfg.Host = conn.Host
		// Reset SSL defaults for new host
		if conn.Host != "localhost" && conn.Host != "127.0.0.1" {
			cfg.SSL = true
			cfg.Port = 443
		} else {
			cfg.SSL = false
			cfg.Port = 8080
		}
	}
	if conn.Port != 0 {
		cfg.Port = conn.Port
	}
	if conn.User != "" {
		cfg.User = conn.User
	}
	if conn.Password != "" {
		cfg.Password = conn.Password
	}
	if conn.Catalog != "" {
		cfg.Catalog = conn.Catalog
	}
	if conn.Schema != "" {
		cfg.Schema = conn.Schema
	}
	if conn.SSL != nil {
		cfg.SSL = *conn.SSL
	}
	if conn.SSLVerify != nil {
		cfg.SSLVerify = *conn.SSLVerify
	}

	return cfg, nil
}

// ConnectionNames returns the names of all available connections.
// Always includes the primary connection name as the first entry.
func (c Config) ConnectionNames() []string {
	names := []string{c.Default}
	for name := range c.Connections {
		names = append(names, name)
	}
	return names
}

// ConnectionCount returns the total number of connections (including default).
func (c Config) ConnectionCount() int {
	return 1 + len(c.Connections)
}

// ConnectionInfo holds display information about a connection.
type ConnectionInfo struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Catalog   string `json:"catalog"`
	Schema    string `json:"schema"`
	SSL       bool   `json:"ssl"`
	IsDefault bool   `json:"is_default"`
}

// ConnectionInfos returns information about all connections for display.
func (c Config) ConnectionInfos() []ConnectionInfo {
	infos := make([]ConnectionInfo, 0, c.ConnectionCount())

	// Add primary connection
	infos = append(infos, ConnectionInfo{
		Name:      c.Default,
		Host:      c.Primary.Host,
		Port:      c.Primary.Port,
		Catalog:   c.Primary.Catalog,
		Schema:    c.Primary.Schema,
		SSL:       c.Primary.SSL,
		IsDefault: true,
	})

	// Add additional connections
	for name := range c.Connections {
		cfg, err := c.ClientConfig(name)
		if err != nil {
			continue // Should not happen for known connections
		}
		infos = append(infos, ConnectionInfo{
			Name:      name,
			Host:      cfg.Host,
			Port:      cfg.Port,
			Catalog:   cfg.Catalog,
			Schema:    cfg.Schema,
			SSL:       cfg.SSL,
			IsDefault: false,
		})
	}

	return infos
}
