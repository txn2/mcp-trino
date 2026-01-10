package extensions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/tools"
)

const (
	// LogKeyRequestID is the context key for the request ID.
	LogKeyRequestID = "request_id"
)

// LoggingMiddleware provides structured logging for tool calls.
// It demonstrates using ToolContext.Set/Get to pass data between Before and After hooks.
type LoggingMiddleware struct {
	logger *slog.Logger
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(w io.Writer) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// Before logs the start of tool execution and generates a request ID.
func (lm *LoggingMiddleware) Before(ctx context.Context, tc *tools.ToolContext) (context.Context, error) {
	// Generate a request ID for correlation
	requestID := generateRequestID()
	tc.Set(LogKeyRequestID, requestID)

	lm.logger.InfoContext(ctx, "tool_call_start",
		slog.String("request_id", requestID),
		slog.String("tool", string(tc.Name)),
	)

	return ctx, nil
}

// After logs the completion of tool execution with duration.
func (lm *LoggingMiddleware) After(
	ctx context.Context,
	tc *tools.ToolContext,
	result *mcp.CallToolResult,
	handlerErr error,
) (*mcp.CallToolResult, error) {
	requestID := tc.GetString(LogKeyRequestID)
	duration := tc.Duration()

	attrs := []any{
		slog.String("request_id", requestID),
		slog.String("tool", string(tc.Name)),
		slog.Duration("duration", duration),
	}

	switch {
	case handlerErr != nil:
		attrs = append(attrs, slog.String("error", handlerErr.Error()))
		lm.logger.ErrorContext(ctx, "tool_call_error", attrs...)
	case result != nil && result.IsError:
		// Tool returned an error result (not a Go error)
		lm.logger.WarnContext(ctx, "tool_call_failed", attrs...)
	default:
		lm.logger.InfoContext(ctx, "tool_call_success", attrs...)
	}

	return result, handlerErr
}

// generateRequestID creates a random request ID for log correlation.
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// Verify LoggingMiddleware implements ToolMiddleware.
var _ tools.ToolMiddleware = (*LoggingMiddleware)(nil)
