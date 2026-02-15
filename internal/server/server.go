// Package server provides the default MCP server setup for mcp-trino.
package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/extensions"
	"github.com/txn2/mcp-trino/pkg/multiserver"
	"github.com/txn2/mcp-trino/pkg/semantic"
	"github.com/txn2/mcp-trino/pkg/semantic/providers/static"
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

	// Descriptions provides custom tool descriptions that override defaults.
	// Keys are tool names (e.g., tools.ToolQuery), values are description strings.
	Descriptions map[tools.ToolName]string

	// SemanticProvider is an optional semantic metadata provider.
	// If nil and SEMANTIC_FILE env var is set, a static provider will be created.
	SemanticProvider semantic.Provider

	// SemanticCacheConfig configures caching for the semantic provider.
	// If nil, default caching (5 minute TTL) is applied when a provider is configured.
	SemanticCacheConfig *semantic.CacheConfig
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
// The server starts even if unconfigured - tools will return helpful errors.
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

	// Check configuration but don't fail - store error for tools to report
	var configErr error
	if err := msCfg.Primary.Validate(); err != nil {
		configErr = fmt.Errorf("trino connection not configured: %w - please configure the extension in Claude Desktop settings", err)
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

	// Apply description overrides if provided
	if len(opts.Descriptions) > 0 {
		toolkitOpts = append(toolkitOpts, tools.WithDescriptions(opts.Descriptions))
	}

	// If unconfigured, add middleware that returns helpful error for all tools
	if configErr != nil {
		toolkitOpts = append(toolkitOpts, tools.WithMiddleware(
			tools.BeforeFunc(func(_ context.Context, _ *tools.ToolContext) (context.Context, error) {
				return nil, configErr
			}),
		))
	}

	// Setup semantic provider
	semanticProvider := opts.SemanticProvider
	if semanticProvider == nil {
		// Check for SEMANTIC_FILE environment variable
		if semanticFile := os.Getenv("SEMANTIC_FILE"); semanticFile != "" {
			provider, err := static.New(static.Config{
				FilePath:      semanticFile,
				WatchInterval: 30 * time.Second, // Enable hot-reload
			})
			if err != nil {
				log.Printf("Warning: Failed to load semantic file %s: %v", semanticFile, err)
			} else {
				semanticProvider = provider
				log.Printf("Loaded semantic metadata from %s", semanticFile)
			}
		}
	}

	// Add semantic provider to toolkit options if configured
	if semanticProvider != nil {
		toolkitOpts = append(toolkitOpts, tools.WithSemanticProvider(semanticProvider))

		// Apply caching
		cacheConfig := opts.SemanticCacheConfig
		if cacheConfig == nil {
			// Default: 5 minute TTL, 10000 entries
			cacheConfig = &semantic.CacheConfig{
				TTL:        5 * time.Minute,
				MaxEntries: 10000,
			}
		}
		toolkitOpts = append(toolkitOpts, tools.WithSemanticCache(*cacheConfig))
	}

	// Create toolkit with multi-server manager and register tools
	toolkit := tools.NewToolkitWithManager(mgr, opts.ToolkitConfig, toolkitOpts...)
	toolkit.RegisterAll(server)

	return server, mgr, nil
}
