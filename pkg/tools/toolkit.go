// Package tools provides MCP tool definitions for Trino operations.
package tools

import (
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// Config configures the Trino toolkit behavior.
type Config struct {
	// DefaultLimit is the default row limit for queries. Default: 1000.
	DefaultLimit int

	// MaxLimit is the maximum allowed row limit. Default: 10000.
	MaxLimit int

	// DefaultTimeout is the default query timeout. Default: 120s.
	DefaultTimeout time.Duration

	// MaxTimeout is the maximum allowed query timeout. Default: 300s.
	MaxTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		DefaultLimit:   1000,
		MaxLimit:       10000,
		DefaultTimeout: 120 * time.Second,
		MaxTimeout:     300 * time.Second,
	}
}

// Toolkit provides MCP tools for Trino operations.
// It's designed to be composable - you can add its tools to any MCP server.
type Toolkit struct {
	client *client.Client
	config Config
}

// NewToolkit creates a new Trino toolkit.
func NewToolkit(c *client.Client, cfg Config) *Toolkit {
	if cfg.DefaultLimit <= 0 {
		cfg.DefaultLimit = 1000
	}
	if cfg.MaxLimit <= 0 {
		cfg.MaxLimit = 10000
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 120 * time.Second
	}
	if cfg.MaxTimeout <= 0 {
		cfg.MaxTimeout = 300 * time.Second
	}

	return &Toolkit{
		client: c,
		config: cfg,
	}
}

// RegisterAll adds all Trino tools to the given MCP server.
// This is the primary composition method - use it to add Trino capabilities
// to your own MCP server.
func (t *Toolkit) RegisterAll(server *mcp.Server) {
	t.registerQueryTool(server)
	t.registerExplainTool(server)
	t.registerListCatalogsTool(server)
	t.registerListSchemasTool(server)
	t.registerListTablesTool(server)
	t.registerDescribeTableTool(server)
}

// Client returns the underlying Trino client.
func (t *Toolkit) Client() *client.Client {
	return t.client
}

// Config returns the toolkit configuration.
func (t *Toolkit) Config() Config {
	return t.config
}
