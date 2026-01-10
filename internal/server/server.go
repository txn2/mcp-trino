// Package server provides the default MCP server setup for mcp-trino.
package server

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/extensions"
	"github.com/txn2/mcp-trino/pkg/tools"
)

// Version is the MCP server version.
const Version = "0.1.0"

// Options configures the server.
type Options struct {
	// ClientConfig is the Trino client configuration.
	ClientConfig client.Config

	// ToolkitConfig is the toolkit configuration.
	ToolkitConfig tools.Config

	// ExtensionsConfig configures middleware, interceptors, and transformers.
	ExtensionsConfig extensions.Config
}

// DefaultOptions returns default server options.
func DefaultOptions() Options {
	return Options{
		ClientConfig:     client.FromEnv(),
		ToolkitConfig:    tools.DefaultConfig(),
		ExtensionsConfig: extensions.FromEnv(),
	}
}

// New creates a new MCP server with Trino tools.
func New(opts Options) (*mcp.Server, *client.Client, error) {
	// Create Trino client
	trinoClient, err := client.New(opts.ClientConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Trino client: %w", err)
	}

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-trino",
		Version: Version,
	}, nil)

	// Build toolkit options from extensions configuration
	toolkitOpts := extensions.BuildToolkitOptions(opts.ExtensionsConfig)

	// Create toolkit and register tools
	toolkit := tools.NewToolkit(trinoClient, opts.ToolkitConfig, toolkitOpts...)
	toolkit.RegisterAll(server)

	return server, trinoClient, nil
}
