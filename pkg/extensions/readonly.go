package extensions

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// ErrModificationBlocked is returned when a modification statement is blocked.
var ErrModificationBlocked = errors.New("modification statements are not allowed in read-only mode")

// ReadOnlyInterceptor blocks modification statements.
// It rejects INSERT, UPDATE, DELETE, DROP, CREATE, ALTER, TRUNCATE, and GRANT statements.
type ReadOnlyInterceptor struct {
	// blockedPatterns matches SQL statements that modify data or schema
	blockedPatterns []*regexp.Regexp
}

// NewReadOnlyInterceptor creates a new read-only interceptor.
func NewReadOnlyInterceptor() *ReadOnlyInterceptor {
	// Patterns to detect modification statements
	// These match at the beginning of the statement (after optional whitespace)
	patterns := []string{
		`(?i)^\s*INSERT\s+`,
		`(?i)^\s*UPDATE\s+`,
		`(?i)^\s*DELETE\s+`,
		`(?i)^\s*DROP\s+`,
		`(?i)^\s*CREATE\s+`,
		`(?i)^\s*ALTER\s+`,
		`(?i)^\s*TRUNCATE\s+`,
		`(?i)^\s*GRANT\s+`,
		`(?i)^\s*REVOKE\s+`,
		`(?i)^\s*MERGE\s+`,
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		compiled[i] = regexp.MustCompile(p)
	}

	return &ReadOnlyInterceptor{
		blockedPatterns: compiled,
	}
}

// Intercept checks if the SQL is a modification statement and blocks it.
func (ri *ReadOnlyInterceptor) Intercept(_ context.Context, sql string, toolName tools.ToolName) (string, error) {
	// Only check query and explain tools
	if toolName != tools.ToolQuery && toolName != tools.ToolExplain {
		return sql, nil
	}

	// Check for blocked patterns
	trimmed := strings.TrimSpace(sql)
	for _, pattern := range ri.blockedPatterns {
		if pattern.MatchString(trimmed) {
			return "", ErrModificationBlocked
		}
	}

	return sql, nil
}

// Verify ReadOnlyInterceptor implements QueryInterceptor.
var _ tools.QueryInterceptor = (*ReadOnlyInterceptor)(nil)
