package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/trinodb/trino-go-client/trino"

	"github.com/txn2/mcp-trino/pkg/client"
)

// failingClient returns a mock whose every query fails with err.
func failingClient(err error) *MockTrinoClient {
	mock := NewMockTrinoClient()
	mock.QueryFunc = func(context.Context, string, client.QueryOptions) (*client.QueryResult, error) {
		return nil, err
	}
	return mock
}

// structuredError calls tool through an MCP client session and returns the
// text content and the "error" object of the structured content, decoded from
// the wire. The error is nil when the result carried none.
func structuredError(t *testing.T, toolkit *Toolkit, tool ToolName, sql string) (text string, errObj map[string]any) {
	t.Helper()
	ctx := context.Background()
	session := connectToolkit(ctx, t, toolkit)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      string(tool),
		Arguments: map[string]any{"sql": sql},
	})
	if err != nil {
		t.Fatalf("CallTool returned a protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("got %d content blocks, want 1", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want *mcp.TextContent", result.Content[0])
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var structured map[string]any
	if err := json.Unmarshal(raw, &structured); err != nil {
		t.Fatalf("structured content %s: %v", raw, err)
	}
	if structured["error"] == nil {
		return tc.Text, nil
	}
	errObj, ok = structured["error"].(map[string]any)
	if !ok {
		t.Fatalf("structured error is %T, want an object", structured["error"])
	}
	return tc.Text, errObj
}

func TestQueryFailure_StructuredErrorOverMCP(t *testing.T) {
	trinoErr := fmt.Errorf("query failed: %w", &trino.ErrQueryFailed{
		StatusCode: 200,
		Reason: &trino.ErrTrino{
			Message: "line 40:10: Table 'scratch.uploads.t' does not exist", ErrorType: "USER_ERROR",
			ErrorName: "TABLE_NOT_FOUND", ErrorCode: 46,
		},
	})
	deadline := fmt.Errorf("query failed: %w", &trino.ErrQueryFailed{Reason: fmt.Errorf(
		"Get \"http://trino/v1/statement/executing/q1/t/112\": %w", context.DeadlineExceeded)})

	tests := []struct {
		name       string
		tool       ToolName
		sql        string
		err        error
		textPrefix string
		want       map[string]any
	}{
		{
			name: "trino_query, error Trino reported", tool: ToolQuery, sql: "SELECT * FROM t",
			err: trinoErr, textPrefix: "Query failed: ",
			want: map[string]any{
				"code": QueryErrorCode, "category": "client_input", "retryable": false,
				"message": "USER_ERROR: line 40:10: Table 'scratch.uploads.t' does not exist",
				"trino": map[string]any{
					"error_type": "USER_ERROR", "error_name": "TABLE_NOT_FOUND",
					"error_code": float64(46), "http_status": float64(200),
				},
				"transport": nil,
			},
		},
		{
			name: "trino_execute, deadline while polling", tool: ToolExecute, sql: "INSERT INTO t VALUES (1)",
			err: deadline, textPrefix: "Execution failed: ",
			want: map[string]any{
				"code": QueryErrorCode, "category": "upstream_unavailable", "retryable": true,
				"message": deadline.Error(),
				"trino":   nil,
				"transport": map[string]any{
					"kind":   "timeout",
					"detail": "Get \"http://trino/v1/statement/executing/q1/t/112\": context deadline exceeded",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolkit := NewToolkit(failingClient(tt.err), DefaultConfig())
			text, got := structuredError(t, toolkit, tt.tool, tt.sql)

			// The text content is what it was before classification existed.
			if want := tt.textPrefix + tt.err.Error(); text != want {
				t.Errorf("text = %q, want %q", text, want)
			}
			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			wantJSON, err := json.Marshal(tt.want)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotJSON, wantJSON) {
				t.Errorf("structuredContent.error =\n %s\nwant\n %s", gotJSON, wantJSON)
			}
		})
	}
}

// TestQueryFailure_ThroughMiddleware checks the classification survives the
// middleware chain, which rebuilds the handler around the base one.
func TestQueryFailure_ThroughMiddleware(t *testing.T) {
	var afterSawError bool
	mw := AfterFunc(func(_ context.Context, _ *ToolContext, result *mcp.CallToolResult, _ error) (*mcp.CallToolResult, error) {
		afterSawError = result != nil && result.IsError
		return result, nil
	})
	toolkit := NewToolkit(failingClient(context.DeadlineExceeded), DefaultConfig(), WithMiddleware(mw))

	_, got := structuredError(t, toolkit, ToolQuery, "SELECT 1")
	if got == nil || got["category"] != "upstream_unavailable" || got["retryable"] != true {
		t.Errorf("structuredContent.error = %v, want retryable upstream_unavailable", got)
	}
	if !afterSawError {
		t.Error("middleware After hook did not see the error result")
	}
}

func TestQueryFailure_CancellationCarriesNoError(t *testing.T) {
	for _, tool := range []ToolName{ToolQuery, ToolExecute} {
		t.Run(string(tool), func(t *testing.T) {
			toolkit := NewToolkit(failingClient(trino.ErrQueryCancelled), DefaultConfig())
			text, got := structuredError(t, toolkit, tool, "SELECT 1")
			if got != nil {
				t.Errorf("structuredContent.error = %v, want none for a canceled query", got)
			}
			if !strings.Contains(text, trino.ErrQueryCancelled.Error()) {
				t.Errorf("text = %q, want the cancellation message", text)
			}
		})
	}
}

func TestFailedQueryOutput_CallerContextEnded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// The client would report the caller's own cancellation as a deadline or
	// cancel; either way the call's context decides, not the error.
	if out := failedQueryOutput(ctx, context.DeadlineExceeded); out != nil {
		t.Errorf("failedQueryOutput = %#v, want nil once the caller's context ended", out)
	}

	out := failedQueryOutput(context.Background(), context.DeadlineExceeded)
	qo, ok := out.(*QueryOutput)
	if !ok || qo.Error == nil || qo.Error.Transport == nil || qo.Error.Transport.Kind != client.TransportTimeout {
		t.Errorf("failedQueryOutput = %#v, want a timeout classification", out)
	}
}
