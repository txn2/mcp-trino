package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolMiddleware intercepts tool execution for cross-cutting concerns
// such as logging, authentication, metrics, rate limiting, etc.
type ToolMiddleware interface {
	// Before is called before tool execution.
	// Return an error to abort execution with an error result.
	// Modify context to pass data to the handler or After hook.
	Before(ctx context.Context, tc *ToolContext) (context.Context, error)

	// After is called after tool execution completes.
	// Can modify the result or handle errors.
	// The handlerErr parameter is the error returned by the handler (if any).
	After(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error)
}

// MiddlewareFunc provides a function-based middleware implementation.
// Use this for simple middleware that doesn't need struct state.
type MiddlewareFunc struct {
	// BeforeFn is called before tool execution.
	BeforeFn func(ctx context.Context, tc *ToolContext) (context.Context, error)

	// AfterFn is called after tool execution.
	AfterFn func(ctx context.Context, tc *ToolContext, result *mcp.CallToolResult, handlerErr error) (*mcp.CallToolResult, error)
}

// Before implements ToolMiddleware.
func (mf MiddlewareFunc) Before(ctx context.Context, tc *ToolContext) (context.Context, error) {
	if mf.BeforeFn != nil {
		return mf.BeforeFn(ctx, tc)
	}
	return ctx, nil
}

// After implements ToolMiddleware.
func (mf MiddlewareFunc) After(
	ctx context.Context,
	tc *ToolContext,
	result *mcp.CallToolResult,
	handlerErr error,
) (*mcp.CallToolResult, error) {
	if mf.AfterFn != nil {
		return mf.AfterFn(ctx, tc, result, handlerErr)
	}
	return result, handlerErr
}

// BeforeFunc creates a MiddlewareFunc with only a Before hook.
func BeforeFunc(fn func(ctx context.Context, tc *ToolContext) (context.Context, error)) MiddlewareFunc {
	return MiddlewareFunc{BeforeFn: fn}
}

// AfterHookFunc is the function signature for After hooks.
type AfterHookFunc func(
	ctx context.Context,
	tc *ToolContext,
	result *mcp.CallToolResult,
	handlerErr error,
) (*mcp.CallToolResult, error)

// AfterFunc creates a MiddlewareFunc with only an After hook.
func AfterFunc(fn AfterHookFunc) MiddlewareFunc {
	return MiddlewareFunc{AfterFn: fn}
}

// MiddlewareChain chains multiple middlewares together.
// Before hooks are executed in order, After hooks in reverse order.
type MiddlewareChain struct {
	middlewares []ToolMiddleware
}

// NewMiddlewareChain creates a middleware chain from the given middlewares.
func NewMiddlewareChain(middlewares ...ToolMiddleware) *MiddlewareChain {
	return &MiddlewareChain{middlewares: middlewares}
}

// Before runs all Before hooks in order.
// Stops and returns error if any middleware returns an error.
func (mc *MiddlewareChain) Before(ctx context.Context, tc *ToolContext) (context.Context, error) {
	var err error
	for _, m := range mc.middlewares {
		ctx, err = m.Before(ctx, tc)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// After runs all After hooks in reverse order.
// This ensures proper unwinding (like defer).
func (mc *MiddlewareChain) After(
	ctx context.Context,
	tc *ToolContext,
	result *mcp.CallToolResult,
	handlerErr error,
) (*mcp.CallToolResult, error) {
	var err error
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		result, err = mc.middlewares[i].After(ctx, tc, result, handlerErr)
		if err != nil {
			// Pass the error to subsequent After hooks
			handlerErr = err
		}
	}
	return result, err
}

// Append adds middlewares to the chain.
func (mc *MiddlewareChain) Append(middlewares ...ToolMiddleware) {
	mc.middlewares = append(mc.middlewares, middlewares...)
}

// Len returns the number of middlewares in the chain.
func (mc *MiddlewareChain) Len() int {
	return len(mc.middlewares)
}
