package tools

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/semantic"
)

func TestWithMiddleware(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	mw := MiddlewareFunc{}
	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithMiddleware(mw))

	if !toolkit.HasMiddleware() {
		t.Error("expected middleware to be configured")
	}
}

func TestWithQueryInterceptor(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql, nil
	})
	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithQueryInterceptor(interceptor))

	if !toolkit.HasInterceptors() {
		t.Error("expected interceptors to be configured")
	}
}

func TestWithResultTransformer(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	transformer := ResultTransformerFunc(func(_ context.Context, _ ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		return result, nil
	})
	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithResultTransformer(transformer))

	if !toolkit.HasTransformers() {
		t.Error("expected transformers to be configured")
	}
}

func TestWithToolMiddleware(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	mw := MiddlewareFunc{}
	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithToolMiddleware(ToolQuery, mw))

	if !toolkit.HasMiddleware() {
		t.Error("expected middleware to be configured")
	}
}

func TestWithPerToolMiddleware(t *testing.T) {
	cfg := &toolConfig{}
	mw := MiddlewareFunc{}

	opt := WithPerToolMiddleware(mw)
	opt(cfg)

	if len(cfg.middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(cfg.middlewares))
	}
}

func TestMultipleOptions(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	mw1 := MiddlewareFunc{}
	mw2 := MiddlewareFunc{}
	interceptor := QueryInterceptorFunc(func(_ context.Context, sql string, _ ToolName) (string, error) {
		return sql, nil
	})
	transformer := ResultTransformerFunc(func(_ context.Context, _ ToolName, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
		return result, nil
	})

	toolkit := NewToolkit(trinoClient, DefaultConfig(),
		WithMiddleware(mw1),
		WithMiddleware(mw2),
		WithQueryInterceptor(interceptor),
		WithResultTransformer(transformer),
		WithToolMiddleware(ToolQuery, mw1),
	)

	if !toolkit.HasMiddleware() {
		t.Error("expected middleware to be configured")
	}
	if !toolkit.HasInterceptors() {
		t.Error("expected interceptors to be configured")
	}
	if !toolkit.HasTransformers() {
		t.Error("expected transformers to be configured")
	}
}

func TestWithSemanticProvider(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	// Create a mock semantic provider
	mockProvider := &semantic.ProviderFunc{
		NameFn: func() string { return "test" },
	}

	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithSemanticProvider(mockProvider))

	if !toolkit.HasSemanticProvider() {
		t.Error("expected semantic provider to be configured")
	}
	if toolkit.SemanticProvider() == nil {
		t.Error("expected SemanticProvider() to return non-nil")
	}
	if toolkit.SemanticProvider().Name() != "test" {
		t.Errorf("expected provider name 'test', got %q", toolkit.SemanticProvider().Name())
	}
}

func TestWithSemanticCache(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	// Create a mock semantic provider
	mockProvider := &semantic.ProviderFunc{
		NameFn: func() string { return "test" },
	}

	cacheConfig := semantic.CacheConfig{
		TTL:        5 * time.Minute,
		MaxEntries: 1000,
	}

	toolkit := NewToolkit(trinoClient, DefaultConfig(),
		WithSemanticProvider(mockProvider),
		WithSemanticCache(cacheConfig),
	)

	if !toolkit.HasSemanticProvider() {
		t.Error("expected semantic provider to be configured")
	}
	// When caching is enabled, the provider is wrapped
	if toolkit.SemanticProvider() == nil {
		t.Error("expected SemanticProvider() to return non-nil")
	}
}

func TestWithSemanticCache_WithoutProvider(t *testing.T) {
	cfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, cfg)

	// Apply cache config without provider - should be a no-op
	cacheConfig := semantic.CacheConfig{
		TTL:        5 * time.Minute,
		MaxEntries: 1000,
	}

	toolkit := NewToolkit(trinoClient, DefaultConfig(),
		WithSemanticCache(cacheConfig),
	)

	if toolkit.HasSemanticProvider() {
		t.Error("expected no semantic provider without WithSemanticProvider")
	}
}
