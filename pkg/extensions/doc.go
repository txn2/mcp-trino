// Package extensions provides production-ready middleware, interceptors, and
// transformers for the mcp-trino toolkit.
//
// This package demonstrates all composability features of the toolkit and
// provides reusable implementations that can be enabled via environment
// variables or programmatic configuration.
//
// # Middleware
//
// Middleware wraps tool execution with Before/After hooks:
//
//   - [LoggingMiddleware]: Structured logging of tool calls with duration tracking
//   - [MetricsMiddleware]: Metrics collection with pluggable backends
//
// # Query Interceptors
//
// Interceptors transform or validate SQL before execution:
//
//   - [ReadOnlyInterceptor]: Blocks modification statements (INSERT, UPDATE, DELETE, etc.)
//   - [QueryLogInterceptor]: Logs all SQL queries for audit/debugging
//
// # Result Transformers
//
// Transformers post-process tool results:
//
//   - [MetadataEnricher]: Adds execution metadata footer to results
//   - [ErrorEnricher]: Adds helpful hints to error messages
//
// # Configuration
//
// Use [FromEnv] to load configuration from environment variables:
//
//	cfg := extensions.FromEnv()
//	opts := extensions.BuildToolkitOptions(cfg)
//	toolkit := tools.NewToolkit(client, toolsCfg, opts...)
//
// # Environment Variables
//
//	MCP_TRINO_EXT_LOGGING   - Enable structured logging (default: false)
//	MCP_TRINO_EXT_METRICS   - Enable metrics collection (default: false)
//	MCP_TRINO_EXT_READONLY  - Block modification statements (default: true)
//	MCP_TRINO_EXT_QUERYLOG  - Log all SQL queries (default: false)
//	MCP_TRINO_EXT_METADATA  - Add execution metadata to results (default: false)
//	MCP_TRINO_EXT_ERRORS    - Add helpful hints to errors (default: true)
package extensions
