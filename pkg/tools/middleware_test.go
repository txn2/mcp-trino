package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMiddlewareFunc_Before(t *testing.T) {
	called := false
	mw := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			called = true
			return ctx, nil
		},
	}

	tc := NewToolContext(ToolQuery, nil)
	_, err := mw.Before(context.Background(), tc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("BeforeFn was not called")
	}
}

func TestMiddlewareFunc_Before_NilFn(t *testing.T) {
	mw := MiddlewareFunc{}
	tc := NewToolContext(ToolQuery, nil)
	ctx, err := mw.Before(context.Background(), tc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestMiddlewareFunc_After(t *testing.T) {
	called := false
	mw := MiddlewareFunc{
		AfterFn: func(
			_ context.Context,
			_ *ToolContext,
			result *mcp.CallToolResult,
			_ error,
		) (*mcp.CallToolResult, error) {
			called = true
			return result, nil
		},
	}

	tc := NewToolContext(ToolQuery, nil)
	result := &mcp.CallToolResult{}
	_, err := mw.After(context.Background(), tc, result, nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("AfterFn was not called")
	}
}

func TestMiddlewareFunc_After_NilFn(t *testing.T) {
	mw := MiddlewareFunc{}
	tc := NewToolContext(ToolQuery, nil)
	result := &mcp.CallToolResult{}
	returnedResult, err := mw.After(context.Background(), tc, result, nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if returnedResult != result {
		t.Error("result should be returned unchanged")
	}
}

func TestBeforeFunc(t *testing.T) {
	called := false
	mw := BeforeFunc(func(ctx context.Context, _ *ToolContext) (context.Context, error) {
		called = true
		return ctx, nil
	})

	tc := NewToolContext(ToolQuery, nil)
	_, err := mw.Before(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("before function was not called")
	}
}

func TestAfterFunc(t *testing.T) {
	called := false
	mw := AfterFunc(func(
		_ context.Context,
		_ *ToolContext,
		result *mcp.CallToolResult,
		_ error,
	) (*mcp.CallToolResult, error) {
		called = true
		return result, nil
	})

	tc := NewToolContext(ToolQuery, nil)
	_, err := mw.After(context.Background(), tc, &mcp.CallToolResult{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("after function was not called")
	}
}

func TestMiddlewareChain_Before(t *testing.T) {
	order := []string{}

	mw1 := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			order = append(order, "mw1")
			return ctx, nil
		},
	}
	mw2 := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			order = append(order, "mw2")
			return ctx, nil
		},
	}

	chain := NewMiddlewareChain(mw1, mw2)
	tc := NewToolContext(ToolQuery, nil)
	_, err := chain.Before(context.Background(), tc)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Errorf("middleware executed in wrong order: %v", order)
	}
}

func TestMiddlewareChain_Before_Error(t *testing.T) {
	expectedErr := errors.New("before error")

	mw1 := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			return ctx, expectedErr
		},
	}
	mw2 := MiddlewareFunc{
		BeforeFn: func(ctx context.Context, _ *ToolContext) (context.Context, error) {
			t.Error("mw2 should not be called")
			return ctx, nil
		},
	}

	chain := NewMiddlewareChain(mw1, mw2)
	tc := NewToolContext(ToolQuery, nil)
	_, err := chain.Before(context.Background(), tc)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestMiddlewareChain_After(t *testing.T) {
	order := []string{}

	mw1 := MiddlewareFunc{
		AfterFn: func(
			_ context.Context,
			_ *ToolContext,
			result *mcp.CallToolResult,
			_ error,
		) (*mcp.CallToolResult, error) {
			order = append(order, "mw1")
			return result, nil
		},
	}
	mw2 := MiddlewareFunc{
		AfterFn: func(
			_ context.Context,
			_ *ToolContext,
			result *mcp.CallToolResult,
			_ error,
		) (*mcp.CallToolResult, error) {
			order = append(order, "mw2")
			return result, nil
		},
	}

	chain := NewMiddlewareChain(mw1, mw2)
	tc := NewToolContext(ToolQuery, nil)
	_, err := chain.After(context.Background(), tc, &mcp.CallToolResult{}, nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// After hooks should run in reverse order.
	if len(order) != 2 || order[0] != "mw2" || order[1] != "mw1" {
		t.Errorf("middleware executed in wrong order (expected reverse): %v", order)
	}
}

func TestMiddlewareChain_Append(t *testing.T) {
	chain := NewMiddlewareChain()
	if chain.Len() != 0 {
		t.Errorf("expected empty chain, got %d", chain.Len())
	}

	mw := MiddlewareFunc{}
	chain.Append(mw)

	if chain.Len() != 1 {
		t.Errorf("expected chain length 1, got %d", chain.Len())
	}
}
