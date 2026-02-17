package extensions

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/tools"
)

// ============================================================================
// Config Tests
// ============================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default values
	if cfg.EnableLogging {
		t.Error("EnableLogging should be false by default")
	}
	if cfg.EnableMetrics {
		t.Error("EnableMetrics should be false by default")
	}
	if !cfg.EnableReadOnly {
		t.Error("EnableReadOnly should be true by default")
	}
	if cfg.EnableQueryLog {
		t.Error("EnableQueryLog should be false by default")
	}
	if cfg.EnableMetadata {
		t.Error("EnableMetadata should be false by default")
	}
	if !cfg.EnableErrorHelp {
		t.Error("EnableErrorHelp should be true by default")
	}
	if cfg.LogOutput != os.Stderr {
		t.Error("LogOutput should default to os.Stderr")
	}
}

func TestFromEnv(t *testing.T) {
	// Save and restore env
	envVars := []string{
		"MCP_TRINO_EXT_LOGGING",
		"MCP_TRINO_EXT_METRICS",
		"MCP_TRINO_EXT_READONLY",
		"MCP_TRINO_EXT_QUERYLOG",
		"MCP_TRINO_EXT_METADATA",
		"MCP_TRINO_EXT_ERRORS",
	}
	savedEnv := make(map[string]string)
	for _, key := range envVars {
		savedEnv[key] = os.Getenv(key)
	}
	defer func() {
		for key, val := range savedEnv {
			if val == "" {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, val)
			}
		}
	}()

	// Clear env
	for _, key := range envVars {
		_ = os.Unsetenv(key)
	}

	// Test default (env not set)
	cfg := FromEnv()
	if cfg.EnableLogging {
		t.Error("EnableLogging should be false when env not set")
	}
	if !cfg.EnableReadOnly {
		t.Error("EnableReadOnly should default to true when env not set")
	}

	// Test with env set
	_ = os.Setenv("MCP_TRINO_EXT_LOGGING", "true")
	_ = os.Setenv("MCP_TRINO_EXT_METRICS", "1")
	_ = os.Setenv("MCP_TRINO_EXT_READONLY", "false")
	_ = os.Setenv("MCP_TRINO_EXT_QUERYLOG", "yes")
	_ = os.Setenv("MCP_TRINO_EXT_METADATA", "on")
	_ = os.Setenv("MCP_TRINO_EXT_ERRORS", "enabled")

	cfg = FromEnv()
	if !cfg.EnableLogging {
		t.Error("EnableLogging should be true when set to 'true'")
	}
	if !cfg.EnableMetrics {
		t.Error("EnableMetrics should be true when set to '1'")
	}
	if cfg.EnableReadOnly {
		t.Error("EnableReadOnly should be false when set to 'false'")
	}
	if !cfg.EnableQueryLog {
		t.Error("EnableQueryLog should be true when set to 'yes'")
	}
	if !cfg.EnableMetadata {
		t.Error("EnableMetadata should be true when set to 'on'")
	}
	if !cfg.EnableErrorHelp {
		t.Error("EnableErrorHelp should be true when set to 'enabled'")
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"enabled", true},
		{"ENABLED", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"disabled", false},
		{"", false},
		{"anything", false},
		{"  true  ", true}, // Whitespace handling
	}

	for _, test := range tests {
		result := parseBool(test.input)
		if result != test.expected {
			t.Errorf("parseBool(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestBuildToolkitOptions(t *testing.T) {
	// Test with all features enabled
	cfg := Config{
		EnableLogging:   true,
		EnableMetrics:   true,
		EnableReadOnly:  true,
		EnableQueryLog:  true,
		EnableMetadata:  true,
		EnableErrorHelp: true,
		LogOutput:       &bytes.Buffer{},
	}

	opts := BuildToolkitOptions(cfg)
	if len(opts) != 6 {
		t.Errorf("Expected 6 options, got %d", len(opts))
	}

	// Test with all features disabled
	cfg = Config{
		EnableLogging:   false,
		EnableMetrics:   false,
		EnableReadOnly:  false,
		EnableQueryLog:  false,
		EnableMetadata:  false,
		EnableErrorHelp: false,
	}

	opts = BuildToolkitOptions(cfg)
	if len(opts) != 0 {
		t.Errorf("Expected 0 options, got %d", len(opts))
	}
}

// ============================================================================
// Logging Middleware Tests
// ============================================================================

func TestLoggingMiddleware_Before(t *testing.T) {
	var buf bytes.Buffer
	lm := NewLoggingMiddleware(&buf)

	tc := tools.NewToolContext(tools.ToolQuery, nil)
	ctx, err := lm.Before(context.Background(), tc)

	if err != nil {
		t.Fatalf("Before returned error: %v", err)
	}
	if ctx == nil {
		t.Fatal("Before returned nil context")
	}

	// Check that request_id was set
	reqID := tc.GetString(LogKeyRequestID)
	if reqID == "" {
		t.Error("request_id should be set in ToolContext")
	}

	// Check log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "tool_call_start") {
		t.Error("Log should contain 'tool_call_start'")
	}
	if !strings.Contains(logOutput, "trino_query") {
		t.Error("Log should contain tool name")
	}
}

func TestLoggingMiddleware_After_Success(t *testing.T) {
	var buf bytes.Buffer
	lm := NewLoggingMiddleware(&buf)

	tc := tools.NewToolContext(tools.ToolQuery, nil)
	tc.Set(LogKeyRequestID, "test-request-id")

	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "success"}},
	}

	returnedResult, err := lm.After(context.Background(), tc, result, nil)

	if err != nil {
		t.Fatalf("After returned error: %v", err)
	}
	if returnedResult != result {
		t.Error("After should return the same result")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "tool_call_success") {
		t.Error("Log should contain 'tool_call_success'")
	}
}

func TestLoggingMiddleware_After_Error(t *testing.T) {
	var buf bytes.Buffer
	lm := NewLoggingMiddleware(&buf)

	tc := tools.NewToolContext(tools.ToolQuery, nil)
	tc.Set(LogKeyRequestID, "test-request-id")

	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "error"}},
		IsError: true,
	}

	_, err := lm.After(context.Background(), tc, result, nil)
	if err != nil {
		t.Fatalf("After returned error: %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "tool_call_failed") {
		t.Error("Log should contain 'tool_call_failed' for error results")
	}
}

// ============================================================================
// Metrics Middleware Tests
// ============================================================================

func TestMetricsMiddleware(t *testing.T) {
	collector := NewInMemoryCollector()
	mm := NewMetricsMiddleware(collector)

	tc := tools.NewToolContext(tools.ToolQuery, nil)

	// Before should be a no-op
	ctx, err := mm.Before(context.Background(), tc)
	if err != nil {
		t.Fatalf("Before returned error: %v", err)
	}
	if ctx == nil {
		t.Fatal("Before returned nil context")
	}

	// Simulate some execution time
	time.Sleep(10 * time.Millisecond)

	// After should record metrics
	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "success"}},
	}

	_, err = mm.After(ctx, tc, result, nil)
	if err != nil {
		t.Fatalf("After returned error: %v", err)
	}

	// Check counter
	count := collector.GetCounter("mcp_trino_tool_calls_total", map[string]string{
		"tool":   "trino_query",
		"status": "success",
	})
	if count != 1 {
		t.Errorf("Expected counter to be 1, got %d", count)
	}

	// Check durations
	durations := collector.GetDurations("mcp_trino_tool_duration_seconds", map[string]string{
		"tool":   "trino_query",
		"status": "success",
	})
	if len(durations) != 1 {
		t.Errorf("Expected 1 duration, got %d", len(durations))
	}
	if durations[0] < 10*time.Millisecond {
		t.Errorf("Duration should be at least 10ms, got %v", durations[0])
	}
}

func TestInMemoryCollector_Reset(t *testing.T) {
	collector := NewInMemoryCollector()
	labels := map[string]string{"test": "true"}

	collector.IncCounter("test_counter", labels)
	collector.ObserveDuration("test_duration", time.Second, labels)

	if collector.GetCounter("test_counter", labels) == 0 {
		t.Error("Counter should be non-zero before reset")
	}

	collector.Reset()

	if collector.GetCounter("test_counter", labels) != 0 {
		t.Error("Counter should be zero after reset")
	}
	if len(collector.GetDurations("test_duration", labels)) != 0 {
		t.Error("Durations should be empty after reset")
	}
}

// ============================================================================
// ReadOnly Interceptor Tests
// ============================================================================

func TestReadOnlyInterceptor_AllowsSelect(t *testing.T) {
	ri := NewReadOnlyInterceptor()

	tests := []string{
		"SELECT * FROM table",
		"select id from users",
		"  SELECT COUNT(*) FROM orders",
		"EXPLAIN SELECT * FROM table",
		"SHOW TABLES",
		"DESCRIBE table",
	}

	for _, sql := range tests {
		result, err := ri.Intercept(context.Background(), sql, tools.ToolQuery)
		if err != nil {
			t.Errorf("Intercept(%q) returned error: %v", sql, err)
		}
		if result != sql {
			t.Errorf("Intercept(%q) = %q, expected unchanged", sql, result)
		}
	}
}

func TestReadOnlyInterceptor_BlocksModifications(t *testing.T) {
	ri := NewReadOnlyInterceptor()

	tests := []string{
		"INSERT INTO table VALUES (1)",
		"UPDATE table SET col = 1",
		"DELETE FROM table",
		"DROP TABLE table",
		"CREATE TABLE table (id INT)",
		"ALTER TABLE table ADD col INT",
		"TRUNCATE TABLE table",
		"GRANT SELECT ON table TO user",
		"REVOKE SELECT ON table FROM user",
		"MERGE INTO target USING source ON ...",
		"  INSERT INTO table VALUES (1)", // With leading whitespace
		"insert into table values (1)",   // Lowercase
	}

	for _, sql := range tests {
		_, err := ri.Intercept(context.Background(), sql, tools.ToolQuery)
		if err == nil {
			t.Errorf("Intercept(%q) should have returned error", sql)
		}
		if !errors.Is(err, ErrModificationBlocked) {
			t.Errorf("Intercept(%q) error = %v, expected ErrModificationBlocked", sql, err)
		}
	}
}

func TestReadOnlyInterceptor_IgnoresNonQueryTools(t *testing.T) {
	ri := NewReadOnlyInterceptor()

	// Should not block for list tools (they don't execute SQL)
	sql := "INSERT INTO table VALUES (1)"
	result, err := ri.Intercept(context.Background(), sql, tools.ToolListTables)
	if err != nil {
		t.Errorf("Should not block for ToolListTables: %v", err)
	}
	if result != sql {
		t.Error("Should return SQL unchanged for non-query tools")
	}
}

func TestReadOnlyInterceptor_BlocksExecuteTool(t *testing.T) {
	ri := NewReadOnlyInterceptor()

	// ReadOnlyInterceptor should also block writes via trino_execute when enabled
	sql := "INSERT INTO table VALUES (1)"
	_, err := ri.Intercept(context.Background(), sql, tools.ToolExecute)
	if err == nil {
		t.Error("ReadOnlyInterceptor should block writes on ToolExecute")
	}
	if !errors.Is(err, ErrModificationBlocked) {
		t.Errorf("expected ErrModificationBlocked, got: %v", err)
	}

	// But allows selects via ToolExecute
	selectSQL := "SELECT * FROM table"
	result, err := ri.Intercept(context.Background(), selectSQL, tools.ToolExecute)
	if err != nil {
		t.Errorf("ReadOnlyInterceptor should allow SELECT on ToolExecute: %v", err)
	}
	if result != selectSQL {
		t.Error("Should return SQL unchanged for SELECT")
	}
}

// ============================================================================
// QueryLog Interceptor Tests
// ============================================================================

func TestQueryLogInterceptor(t *testing.T) {
	var buf bytes.Buffer
	ql := NewQueryLogInterceptor(&buf)

	sql := "SELECT * FROM users"
	result, err := ql.Intercept(context.Background(), sql, tools.ToolQuery)

	if err != nil {
		t.Fatalf("Intercept returned error: %v", err)
	}
	if result != sql {
		t.Errorf("Intercept should return SQL unchanged, got %q", result)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "query_intercepted") {
		t.Error("Log should contain 'query_intercepted'")
	}
	if !strings.Contains(logOutput, sql) {
		t.Error("Log should contain the SQL query")
	}
	if !strings.Contains(logOutput, "trino_query") {
		t.Error("Log should contain the tool name")
	}
}

// ============================================================================
// Metadata Enricher Tests
// ============================================================================

func TestMetadataEnricher(t *testing.T) {
	me := NewMetadataEnricher()

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Query result"},
		},
	}

	transformed, err := me.Transform(context.Background(), tools.ToolQuery, result)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	// Check that metadata was appended
	tc, ok := transformed.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content should be TextContent")
	}
	if !strings.Contains(tc.Text, "---") {
		t.Error("Result should contain metadata separator")
	}
	if !strings.Contains(tc.Text, "Tool: trino_query") {
		t.Error("Result should contain tool name")
	}
}

func TestMetadataEnricher_NilResult(t *testing.T) {
	me := NewMetadataEnricher()

	result, err := me.Transform(context.Background(), tools.ToolQuery, nil)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	if result != nil {
		t.Error("Transform should return nil for nil input")
	}
}

func TestMetadataEnricher_NoTextContent(t *testing.T) {
	me := NewMetadataEnricher()

	result := &mcp.CallToolResult{
		Content: []mcp.Content{}, // No content
	}

	transformed, err := me.Transform(context.Background(), tools.ToolQuery, result)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	// Should add new text content
	if len(transformed.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(transformed.Content))
	}
}

// ============================================================================
// Error Enricher Tests
// ============================================================================

func TestErrorEnricher_AddsHints(t *testing.T) {
	ee := NewErrorEnricher()

	tests := []struct {
		errorMsg     string
		expectedHint string
	}{
		{"Table not found: users", "trino_list_tables"},
		{"Schema not found: test", "trino_list_schemas"},
		{"Catalog not found: hive", "trino_list_catalogs"},
		{"Access denied", "permissions"},
		{"Syntax error at line 1", "syntax"},
		{"Column not found: foo", "trino_describe_table"},
	}

	for _, test := range tests {
		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: test.errorMsg}},
			IsError: true,
		}

		transformed, err := ee.Transform(context.Background(), tools.ToolQuery, result)
		if err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		tc, ok := transformed.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("Content should be TextContent")
		}
		if !strings.Contains(strings.ToLower(tc.Text), test.expectedHint) {
			t.Errorf("Error '%s' should have hint containing '%s', got: %s",
				test.errorMsg, test.expectedHint, tc.Text)
		}
	}
}

func TestErrorEnricher_SkipsNonErrors(t *testing.T) {
	ee := NewErrorEnricher()

	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Table not found"}},
		IsError: false, // Not an error
	}

	transformed, err := ee.Transform(context.Background(), tools.ToolQuery, result)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	tc, ok := transformed.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content should be TextContent")
	}
	// Should not have hint since it's not an error
	if strings.Contains(tc.Text, "Hint:") {
		t.Error("Non-error results should not have hints added")
	}
}

func TestErrorEnricher_AddHint(t *testing.T) {
	ee := NewErrorEnricher()
	ee.AddHint("custom error pattern", "Custom hint text")

	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "A Custom Error Pattern occurred"}},
		IsError: true,
	}

	transformed, err := ee.Transform(context.Background(), tools.ToolQuery, result)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	tc, ok := transformed.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content should be TextContent")
	}
	if !strings.Contains(tc.Text, "Custom hint text") {
		t.Error("Custom hint should be added")
	}
}

func TestErrorEnricher_NilResult(t *testing.T) {
	ee := NewErrorEnricher()

	result, err := ee.Transform(context.Background(), tools.ToolQuery, nil)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	if result != nil {
		t.Error("Transform should return nil for nil input")
	}
}

// ============================================================================
// Interface Compliance Tests
// ============================================================================

func TestInterfaceCompliance(t *testing.T) {
	t.Helper()
	// These tests verify at compile time that our types implement the expected interfaces
	var _ tools.ToolMiddleware = (*LoggingMiddleware)(nil)
	var _ tools.ToolMiddleware = (*MetricsMiddleware)(nil)
	var _ tools.QueryInterceptor = (*ReadOnlyInterceptor)(nil)
	var _ tools.QueryInterceptor = (*QueryLogInterceptor)(nil)
	var _ tools.ResultTransformer = (*MetadataEnricher)(nil)
	var _ tools.ResultTransformer = (*ErrorEnricher)(nil)
	var _ MetricsCollector = (*InMemoryCollector)(nil)
}
