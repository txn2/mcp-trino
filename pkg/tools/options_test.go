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

func TestWithDescription(t *testing.T) {
	cfg := &toolConfig{}
	opt := WithDescription("Custom description")
	opt(cfg)

	if cfg.description == nil {
		t.Fatal("expected description to be set")
	}
	if *cfg.description != "Custom description" {
		t.Errorf("expected 'Custom description', got %q", *cfg.description)
	}
}

func TestWithDescriptions(t *testing.T) {
	clientCfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, clientCfg)

	descs := map[ToolName]string{
		ToolQuery:   "Custom query desc",
		ToolExplain: "Custom explain desc",
	}
	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithDescriptions(descs))

	// Verify descriptions were stored
	if toolkit.descriptions[ToolQuery] != "Custom query desc" {
		t.Errorf("expected 'Custom query desc', got %q", toolkit.descriptions[ToolQuery])
	}
	if toolkit.descriptions[ToolExplain] != "Custom explain desc" {
		t.Errorf("expected 'Custom explain desc', got %q", toolkit.descriptions[ToolExplain])
	}
}

func TestWithDescriptions_Merge(t *testing.T) {
	clientCfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, clientCfg)

	// Apply two rounds of WithDescriptions — second should merge, not replace
	toolkit := NewToolkit(trinoClient, DefaultConfig(),
		WithDescriptions(map[ToolName]string{
			ToolQuery: "First query desc",
		}),
		WithDescriptions(map[ToolName]string{
			ToolExplain: "Explain desc",
		}),
	)

	if toolkit.descriptions[ToolQuery] != "First query desc" {
		t.Errorf("expected 'First query desc', got %q", toolkit.descriptions[ToolQuery])
	}
	if toolkit.descriptions[ToolExplain] != "Explain desc" {
		t.Errorf("expected 'Explain desc', got %q", toolkit.descriptions[ToolExplain])
	}
}

func TestWithAnnotation(t *testing.T) {
	cfg := &toolConfig{}
	ann := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	opt := WithAnnotation(ann)
	opt(cfg)

	if cfg.annotations == nil {
		t.Fatal("expected annotations to be set")
	}
	if !cfg.annotations.ReadOnlyHint {
		t.Error("expected ReadOnlyHint=true")
	}
	if !cfg.annotations.IdempotentHint {
		t.Error("expected IdempotentHint=true")
	}
}

func TestWithAnnotations(t *testing.T) {
	clientCfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, clientCfg)

	anns := map[ToolName]*mcp.ToolAnnotations{
		ToolQuery:   {ReadOnlyHint: true},
		ToolExplain: {IdempotentHint: false},
	}
	toolkit := NewToolkit(trinoClient, DefaultConfig(), WithAnnotations(anns))

	if toolkit.annotations[ToolQuery] == nil || !toolkit.annotations[ToolQuery].ReadOnlyHint {
		t.Error("expected ToolQuery annotation ReadOnlyHint=true")
	}
	if toolkit.annotations[ToolExplain] == nil {
		t.Fatal("expected ToolExplain annotation to be set")
	}
	if toolkit.annotations[ToolExplain].IdempotentHint {
		t.Error("expected ToolExplain annotation IdempotentHint=false")
	}
}

func TestWithAnnotations_Merge(t *testing.T) {
	clientCfg := client.Config{
		Host: "localhost",
		User: "test",
	}
	trinoClient := client.NewWithDB(nil, clientCfg)

	toolkit := NewToolkit(trinoClient, DefaultConfig(),
		WithAnnotations(map[ToolName]*mcp.ToolAnnotations{
			ToolQuery: {ReadOnlyHint: true},
		}),
		WithAnnotations(map[ToolName]*mcp.ToolAnnotations{
			ToolExplain: {IdempotentHint: true},
		}),
	)

	if toolkit.annotations[ToolQuery] == nil {
		t.Error("expected ToolQuery annotation to survive merge")
	}
	if toolkit.annotations[ToolExplain] == nil {
		t.Error("expected ToolExplain annotation from second call")
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
