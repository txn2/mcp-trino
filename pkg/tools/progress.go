package tools

import "context"

// ProgressNotifier sends progress notifications during tool execution.
// The platform injects an implementation via ToolMiddleware that bridges
// to the MCP ServerSession.
type ProgressNotifier interface {
	// Notify sends a progress update.
	// progress is the current step, total is the total number of steps.
	// message describes the current operation.
	Notify(ctx context.Context, progress, total float64, message string) error
}

// progressNotifierKey is the context key for ProgressNotifier.
type progressNotifierKey struct{}

// WithProgressNotifier returns a new context with the given ProgressNotifier.
func WithProgressNotifier(ctx context.Context, n ProgressNotifier) context.Context {
	return context.WithValue(ctx, progressNotifierKey{}, n)
}

// GetProgressNotifier retrieves the ProgressNotifier from the context.
// Returns nil if no notifier is set.
func GetProgressNotifier(ctx context.Context) ProgressNotifier {
	n, _ := ctx.Value(progressNotifierKey{}).(ProgressNotifier) //nolint:errcheck // type assertion ok is unused by design
	return n
}
