package extensions

import (
	"io"
	"os"
	"strings"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// Config configures which extensions are enabled.
type Config struct {
	// Middleware
	EnableLogging bool // MCP_TRINO_EXT_LOGGING
	EnableMetrics bool // MCP_TRINO_EXT_METRICS

	// Query Interceptors
	EnableReadOnly bool // MCP_TRINO_EXT_READONLY (default: true)
	EnableQueryLog bool // MCP_TRINO_EXT_QUERYLOG

	// Result Transformers
	EnableMetadata  bool // MCP_TRINO_EXT_METADATA
	EnableErrorHelp bool // MCP_TRINO_EXT_ERRORS (default: true)

	// Output destination for logging middleware and query log
	LogOutput io.Writer
}

// DefaultConfig returns a Config with safe defaults.
// ReadOnly and ErrorHelp are enabled by default for safety.
func DefaultConfig() Config {
	return Config{
		EnableLogging:   false,
		EnableMetrics:   false,
		EnableReadOnly:  true,
		EnableQueryLog:  false,
		EnableMetadata:  false,
		EnableErrorHelp: true,
		LogOutput:       os.Stderr,
	}
}

// FromEnv loads extension configuration from environment variables.
// Uses DefaultConfig as the base and overrides with environment values.
func FromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("MCP_TRINO_EXT_LOGGING"); v != "" {
		cfg.EnableLogging = parseBool(v)
	}
	if v := os.Getenv("MCP_TRINO_EXT_METRICS"); v != "" {
		cfg.EnableMetrics = parseBool(v)
	}
	if v := os.Getenv("MCP_TRINO_EXT_READONLY"); v != "" {
		cfg.EnableReadOnly = parseBool(v)
	}
	if v := os.Getenv("MCP_TRINO_EXT_QUERYLOG"); v != "" {
		cfg.EnableQueryLog = parseBool(v)
	}
	if v := os.Getenv("MCP_TRINO_EXT_METADATA"); v != "" {
		cfg.EnableMetadata = parseBool(v)
	}
	if v := os.Getenv("MCP_TRINO_EXT_ERRORS"); v != "" {
		cfg.EnableErrorHelp = parseBool(v)
	}

	return cfg
}

// BuildToolkitOptions converts the extension configuration into toolkit options.
// This is the primary integration point - pass these options to tools.NewToolkit.
func BuildToolkitOptions(cfg Config) []tools.ToolkitOption {
	var opts []tools.ToolkitOption

	logOutput := cfg.LogOutput
	if logOutput == nil {
		logOutput = os.Stderr
	}

	if cfg.EnableLogging {
		opts = append(opts, tools.WithMiddleware(NewLoggingMiddleware(logOutput)))
	}
	if cfg.EnableMetrics {
		opts = append(opts, tools.WithMiddleware(NewMetricsMiddleware(NewInMemoryCollector())))
	}
	if cfg.EnableReadOnly {
		opts = append(opts, tools.WithQueryInterceptor(NewReadOnlyInterceptor()))
	}
	if cfg.EnableQueryLog {
		opts = append(opts, tools.WithQueryInterceptor(NewQueryLogInterceptor(logOutput)))
	}
	if cfg.EnableMetadata {
		opts = append(opts, tools.WithResultTransformer(NewMetadataEnricher()))
	}
	if cfg.EnableErrorHelp {
		opts = append(opts, tools.WithResultTransformer(NewErrorEnricher()))
	}

	return opts
}

// parseBool parses a string as a boolean, accepting common variations.
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
