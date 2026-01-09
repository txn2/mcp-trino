package tools

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
