package tools

import "github.com/txn2/mcp-trino/pkg/semantic"

// ToolkitOption configures a Toolkit during construction.
// Use with NewToolkit to add middleware, interceptors, and transformers.
type ToolkitOption func(*Toolkit)

// WithMiddleware adds a global middleware applied to all tools.
// Middleware is executed in the order added.
func WithMiddleware(m ToolMiddleware) ToolkitOption {
	return func(t *Toolkit) {
		t.middlewares = append(t.middlewares, m)
	}
}

// WithQueryInterceptor adds a query interceptor for SQL tools.
// Interceptors are executed in the order added.
// Only applies to ToolQuery and ToolExplain.
func WithQueryInterceptor(i QueryInterceptor) ToolkitOption {
	return func(t *Toolkit) {
		t.interceptors = append(t.interceptors, i)
	}
}

// WithResultTransformer adds a result transformer.
// Transformers are executed in the order added, after middleware After hooks.
func WithResultTransformer(tr ResultTransformer) ToolkitOption {
	return func(t *Toolkit) {
		t.transformers = append(t.transformers, tr)
	}
}

// WithToolMiddleware adds middleware for a specific tool.
// This middleware only runs when the named tool is executed.
func WithToolMiddleware(name ToolName, m ToolMiddleware) ToolkitOption {
	return func(t *Toolkit) {
		if t.toolMiddlewares == nil {
			t.toolMiddlewares = make(map[ToolName][]ToolMiddleware)
		}
		t.toolMiddlewares[name] = append(t.toolMiddlewares[name], m)
	}
}

// toolConfig holds per-registration configuration for a tool.
type toolConfig struct {
	middlewares []ToolMiddleware
	description *string
}

// ToolOption configures a single tool during registration.
// Use with RegisterWith for per-registration customization.
type ToolOption func(*toolConfig)

// WithPerToolMiddleware adds middleware only for this specific registration.
// This is useful when registering the same tool type with different configurations.
func WithPerToolMiddleware(m ToolMiddleware) ToolOption {
	return func(tc *toolConfig) {
		tc.middlewares = append(tc.middlewares, m)
	}
}

// WithDescription sets a custom description for a single tool registration.
// Use with RegisterWith for per-registration description override.
//
// Example:
//
//	toolkit.RegisterWith(server, tools.ToolQuery,
//	    tools.WithDescription("Query the retail analytics warehouse"),
//	)
func WithDescription(desc string) ToolOption {
	return func(tc *toolConfig) {
		tc.description = &desc
	}
}

// WithDescriptions sets custom descriptions for multiple tools at the toolkit level.
// These override default descriptions but are themselves overridden by per-registration
// WithDescription calls.
//
// Example:
//
//	toolkit := tools.NewToolkit(client, cfg,
//	    tools.WithDescriptions(map[tools.ToolName]string{
//	        tools.ToolQuery:   "Query the retail analytics warehouse",
//	        tools.ToolExplain: "Check query performance",
//	    }),
//	)
func WithDescriptions(descs map[ToolName]string) ToolkitOption {
	return func(t *Toolkit) {
		for name, desc := range descs {
			t.descriptions[name] = desc
		}
	}
}

// WithSemanticProvider adds a semantic metadata provider to the toolkit.
// When configured, tools like describe_table and list_tables will enrich
// their output with semantic metadata from the provider.
//
// Example:
//
//	datahub := datahub.New(datahub.FromEnv())
//	toolkit := tools.NewToolkit(client, cfg,
//	    tools.WithSemanticProvider(datahub),
//	)
func WithSemanticProvider(provider semantic.Provider) ToolkitOption {
	return func(t *Toolkit) {
		t.semanticProvider = provider
	}
}

// WithSemanticCache wraps the semantic provider with caching.
// Only applies if a semantic provider is configured.
//
// Example:
//
//	toolkit := tools.NewToolkit(client, cfg,
//	    tools.WithSemanticProvider(datahub),
//	    tools.WithSemanticCache(semantic.DefaultCacheConfig()),
//	)
func WithSemanticCache(cfg semantic.CacheConfig) ToolkitOption {
	return func(t *Toolkit) {
		t.semanticCacheConfig = &cfg
	}
}
