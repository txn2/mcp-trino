// Package tools provides MCP tool definitions for Trino operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/multiserver"
	"github.com/txn2/mcp-trino/pkg/semantic"
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
	client  TrinoClient          // Single client mode (for backwards compatibility)
	manager *multiserver.Manager // Multi-server mode (optional)
	config  Config

	// Extensibility hooks (all optional, zero-value = no overhead)
	middlewares     []ToolMiddleware              // Global middleware
	interceptors    []QueryInterceptor            // SQL interceptors
	transformers    []ResultTransformer           // Result transformers
	toolMiddlewares map[ToolName][]ToolMiddleware // Per-tool middleware

	// Semantic layer (optional, zero-overhead if nil)
	semanticProvider    semantic.Provider
	semanticCacheConfig *semantic.CacheConfig

	// Description overrides (toolkit-level)
	descriptions map[ToolName]string

	// Annotation overrides (toolkit-level)
	annotations map[ToolName]*mcp.ToolAnnotations

	// Icon overrides (toolkit-level)
	icons map[ToolName][]mcp.Icon

	// Internal tracking
	registeredTools map[ToolName]bool
}

// NewToolkit creates a new Trino toolkit.
// Accepts optional ToolkitOption arguments for middleware, interceptors, etc.
// Maintains backwards compatibility - existing code works unchanged.
func NewToolkit(c TrinoClient, cfg Config, opts ...ToolkitOption) *Toolkit {
	t := newBaseToolkit(normalizeConfig(cfg))
	t.client = c
	applyToolkitOptions(t, opts)
	return t
}

// normalizeConfig applies default values to a Config.
func normalizeConfig(cfg Config) Config {
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
	return cfg
}

// newBaseToolkit creates a toolkit with common fields initialized.
func newBaseToolkit(cfg Config) *Toolkit {
	return &Toolkit{
		config:          cfg,
		toolMiddlewares: make(map[ToolName][]ToolMiddleware),
		descriptions:    make(map[ToolName]string),
		annotations:     make(map[ToolName]*mcp.ToolAnnotations),
		icons:           make(map[ToolName][]mcp.Icon),
		registeredTools: make(map[ToolName]bool),
	}
}

// applyToolkitOptions applies options and finalizes toolkit setup.
func applyToolkitOptions(t *Toolkit, opts []ToolkitOption) {
	for _, opt := range opts {
		opt(t)
	}

	// Apply caching to semantic provider if configured
	if t.semanticProvider != nil && t.semanticCacheConfig != nil {
		t.semanticProvider = semantic.NewCachingProvider(t.semanticProvider, *t.semanticCacheConfig)
	}
}

// RegisterAll adds all Trino tools to the given MCP server.
// This is the primary composition method - use it to add Trino capabilities
// to your own MCP server.
func (t *Toolkit) RegisterAll(server *mcp.Server) {
	t.Register(server, AllTools()...)
}

// Register adds specific tools to the server.
// Use this for selective tool registration instead of RegisterAll.
//
// Example:
//
//	toolkit.Register(server, tools.ToolQuery, tools.ToolExplain)
func (t *Toolkit) Register(server *mcp.Server, names ...ToolName) {
	for _, name := range names {
		t.registerTool(server, name, nil)
	}
}

// RegisterWith adds a tool with additional per-registration options.
//
// Example:
//
//	toolkit.RegisterWith(server, tools.ToolQuery,
//	    tools.WithPerToolMiddleware(rateLimiter),
//	)
func (t *Toolkit) RegisterWith(server *mcp.Server, name ToolName, opts ...ToolOption) {
	cfg := &toolConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	t.registerTool(server, name, cfg)
}

// registerTool is the internal registration method.
func (t *Toolkit) registerTool(server *mcp.Server, name ToolName, cfg *toolConfig) {
	if t.registeredTools[name] {
		return // Already registered
	}

	switch name {
	case ToolQuery:
		t.registerQueryTool(server, cfg)
	case ToolExecute:
		t.registerExecuteTool(server, cfg)
	case ToolExplain:
		t.registerExplainTool(server, cfg)
	case ToolListCatalogs:
		t.registerListCatalogsTool(server, cfg)
	case ToolListSchemas:
		t.registerListSchemasTool(server, cfg)
	case ToolListTables:
		t.registerListTablesTool(server, cfg)
	case ToolDescribeTable:
		t.registerDescribeTableTool(server, cfg)
	case ToolListConnections:
		t.registerListConnectionsTool(server, cfg)
	}

	t.registeredTools[name] = true
}

// InterceptSQL applies all registered interceptors to a SQL query.
// Call this from custom tool handlers that execute SQL.
//
// Example:
//
//	func myHandler(ctx context.Context, req *mcp.CallToolRequest, input MyInput) (*mcp.CallToolResult, any, error) {
//	    sql, err := toolkit.InterceptSQL(ctx, input.SQL, "my_custom_query")
//	    if err != nil {
//	        return ErrorResult(err.Error()), nil, nil
//	    }
//	    // Execute sql...
//	}
func (t *Toolkit) InterceptSQL(ctx context.Context, sql string, toolName ToolName) (string, error) {
	if len(t.interceptors) == 0 {
		return sql, nil
	}

	var err error
	for _, i := range t.interceptors {
		sql, err = i.Intercept(ctx, sql, toolName)
		if err != nil {
			return "", err
		}
	}
	return sql, nil
}

// wrapHandler wraps a handler with middleware and transformer support.
// Returns the original handler if no middleware is configured (zero overhead).
func (t *Toolkit) wrapHandler(
	name ToolName,
	handler func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error),
	cfg *toolConfig,
) func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
	// Collect all applicable middlewares
	var allMiddlewares []ToolMiddleware
	allMiddlewares = append(allMiddlewares, t.middlewares...)           // Global
	allMiddlewares = append(allMiddlewares, t.toolMiddlewares[name]...) // Per-tool
	if cfg != nil {
		allMiddlewares = append(allMiddlewares, cfg.middlewares...) // Per-registration
	}

	// If no middleware or transformers configured, return handler unchanged
	if len(allMiddlewares) == 0 && len(t.transformers) == 0 {
		return handler
	}

	return func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		tc := NewToolContext(name, input)

		// Run Before hooks
		var err error
		for _, m := range allMiddlewares {
			ctx, err = m.Before(ctx, tc)
			if err != nil {
				return ErrorResult(fmt.Sprintf("middleware error: %v", err)), nil, nil
			}
		}

		// Execute handler
		result, extra, handlerErr := handler(ctx, req, input)

		// Run After hooks (reverse order)
		for i := len(allMiddlewares) - 1; i >= 0; i-- {
			result, err = allMiddlewares[i].After(ctx, tc, result, handlerErr)
			if err != nil {
				return ErrorResult(fmt.Sprintf("middleware error: %v", err)), nil, nil
			}
		}

		// Run transformers
		for _, tr := range t.transformers {
			result, err = tr.Transform(ctx, name, result)
			if err != nil {
				return ErrorResult(fmt.Sprintf("transformer error: %v", err)), nil, nil
			}
		}

		return result, extra, nil
	}
}

// Client returns the underlying Trino client.
func (t *Toolkit) Client() TrinoClient {
	return t.client
}

// Config returns the toolkit configuration.
func (t *Toolkit) Config() Config {
	return t.config
}

// HasMiddleware returns true if any middleware is configured.
func (t *Toolkit) HasMiddleware() bool {
	if len(t.middlewares) > 0 {
		return true
	}
	for _, mws := range t.toolMiddlewares {
		if len(mws) > 0 {
			return true
		}
	}
	return false
}

// HasInterceptors returns true if any query interceptors are configured.
func (t *Toolkit) HasInterceptors() bool {
	return len(t.interceptors) > 0
}

// HasTransformers returns true if any result transformers are configured.
func (t *Toolkit) HasTransformers() bool {
	return len(t.transformers) > 0
}

// NewToolkitWithManager creates a Toolkit with multi-server support.
// Use this when you need to connect to multiple Trino servers.
func NewToolkitWithManager(mgr *multiserver.Manager, cfg Config, opts ...ToolkitOption) *Toolkit {
	t := newBaseToolkit(normalizeConfig(cfg))
	t.manager = mgr
	applyToolkitOptions(t, opts)
	return t
}

// getClient returns the Trino client for the given connection name.
// If connection is empty, returns the default client.
// In single-client mode, always returns the single client.
func (t *Toolkit) getClient(connection string) (TrinoClient, error) {
	// Multi-server mode
	if t.manager != nil {
		return t.manager.Client(connection)
	}

	// Single-client mode - ignore connection parameter
	if t.client == nil {
		return nil, fmt.Errorf("no client configured")
	}
	return t.client, nil
}

// HasManager returns true if multi-server mode is enabled.
func (t *Toolkit) HasManager() bool {
	return t.manager != nil
}

// Manager returns the connection manager, or nil if in single-client mode.
func (t *Toolkit) Manager() *multiserver.Manager {
	return t.manager
}

// ConnectionInfos returns information about all configured connections.
// Returns a single "default" connection in single-client mode.
func (t *Toolkit) ConnectionInfos() []multiserver.ConnectionInfo {
	if t.manager != nil {
		return t.manager.ConnectionInfos()
	}

	// Single-client mode - return default connection info
	return []multiserver.ConnectionInfo{
		{
			Name:      "default",
			Host:      "configured via single client",
			IsDefault: true,
		},
	}
}

// ConnectionCount returns the number of configured connections.
func (t *Toolkit) ConnectionCount() int {
	if t.manager != nil {
		return t.manager.ConnectionCount()
	}
	return 1
}

// SemanticProvider returns the configured semantic provider, or nil if not configured.
func (t *Toolkit) SemanticProvider() semantic.Provider {
	return t.semanticProvider
}

// HasSemanticProvider returns true if a semantic provider is configured.
func (t *Toolkit) HasSemanticProvider() bool {
	return t.semanticProvider != nil
}
