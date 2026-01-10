package extensions

import (
	"context"
	"io"
	"log/slog"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// QueryLogInterceptor logs all SQL queries for audit and debugging.
// It logs the SQL and tool name, then returns the SQL unchanged.
type QueryLogInterceptor struct {
	logger *slog.Logger
}

// NewQueryLogInterceptor creates a new query log interceptor.
func NewQueryLogInterceptor(w io.Writer) *QueryLogInterceptor {
	return &QueryLogInterceptor{
		logger: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// Intercept logs the SQL query and returns it unchanged.
func (ql *QueryLogInterceptor) Intercept(ctx context.Context, sql string, toolName tools.ToolName) (string, error) {
	ql.logger.InfoContext(ctx, "query_intercepted",
		slog.String("tool", string(toolName)),
		slog.String("sql", sql),
	)
	return sql, nil
}

// Verify QueryLogInterceptor implements QueryInterceptor.
var _ tools.QueryInterceptor = (*QueryLogInterceptor)(nil)
