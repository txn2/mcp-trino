package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
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
