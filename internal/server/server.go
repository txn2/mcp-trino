// Package server provides the default MCP server setup for mcp-trino.
package server

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/extensions"
	"github.com/txn2/mcp-trino/pkg/multiserver"
	"github.com/txn2/mcp-trino/pkg/tools"
)

// Version is the MCP server version.
const Version = "0.1.0"

// Options configures the server.
type Options struct {
	// MultiServerConfig is the multi-server configuration.
	// If nil, will be loaded from environment via multiserver.FromEnv().
	MultiServerConfig *multiserver.Config

	// ToolkitConfig is the toolkit configuration.
	ToolkitConfig tools.Config

	// ExtensionsConfig configures middleware, interceptors, and transformers.
	ExtensionsConfig extensions.Config
}

// DefaultOptions returns default server options.
// Note: MultiServerConfig is loaded from environment when nil.
func DefaultOptions() Options {
	return Options{
		MultiServerConfig: nil, // Loaded from env in New()
		ToolkitConfig:     tools.DefaultConfig(),
		ExtensionsConfig:  extensions.FromEnv(),
	}
}

// New creates a new MCP server with Trino tools.
// Returns the MCP server and the connection manager for cleanup.
func New(opts Options) (*mcp.Server, *multiserver.Manager, error) {
	// Load multi-server config from environment if not provided
	var msCfg multiserver.Config
	if opts.MultiServerConfig != nil {
		msCfg = *opts.MultiServerConfig
	} else {
		var err error
		msCfg, err = multiserver.FromEnv()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load server configuration: %w", err)
		}
	}

	// Validate primary configuration
	if err := msCfg.Primary.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid primary configuration: %w", err)
	}

	// Create connection manager
	mgr := multiserver.NewManager(msCfg)

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-trino",
		Version: Version,
	}, nil)

	// Build toolkit options from extensions configuration
	toolkitOpts := extensions.BuildToolkitOptions(opts.ExtensionsConfig)

	// Create toolkit with multi-server manager and register tools
	toolkit := tools.NewToolkitWithManager(mgr, opts.ToolkitConfig, toolkitOpts...)
	toolkit.RegisterAll(server)

	return server, mgr, nil
}
