package tools

import (
	"context"
)

// QueryInterceptor transforms SQL queries before execution.
// Use interceptors for validation, tenant isolation, query rewriting,
// audit logging, or any SQL-specific transformation.
type QueryInterceptor interface {
	// Intercept transforms the SQL query before execution.
	// Return an error to reject the query.
	// The toolName parameter indicates which tool is executing the query.
	Intercept(ctx context.Context, sql string, toolName ToolName) (string, error)
}

// QueryInterceptorFunc allows using a function as an interceptor.
type QueryInterceptorFunc func(ctx context.Context, sql string, toolName ToolName) (string, error)

// Intercept implements QueryInterceptor.
func (f QueryInterceptorFunc) Intercept(ctx context.Context, sql string, toolName ToolName) (string, error) {
	return f(ctx, sql, toolName)
}

// InterceptorChain chains multiple interceptors together.
// Interceptors are executed in order; each receives the output of the previous.
type InterceptorChain struct {
	interceptors []QueryInterceptor
}

// NewInterceptorChain creates an interceptor chain from the given interceptors.
func NewInterceptorChain(interceptors ...QueryInterceptor) *InterceptorChain {
	return &InterceptorChain{interceptors: interceptors}
}

// Intercept runs all interceptors in order.
// Stops and returns error if any interceptor returns an error.
func (ic *InterceptorChain) Intercept(ctx context.Context, sql string, toolName ToolName) (string, error) {
	var err error
	for _, i := range ic.interceptors {
		sql, err = i.Intercept(ctx, sql, toolName)
		if err != nil {
			return "", err
		}
	}
	return sql, nil
}

// Append adds interceptors to the chain.
func (ic *InterceptorChain) Append(interceptors ...QueryInterceptor) {
	ic.interceptors = append(ic.interceptors, interceptors...)
}

// Len returns the number of interceptors in the chain.
func (ic *InterceptorChain) Len() int {
	return len(ic.interceptors)
}
