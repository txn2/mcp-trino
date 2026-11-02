package tools

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// hookLog records, in order, each hook that ran and the tool name it saw.
type hookLog struct {
	mu     sync.Mutex
	events []string
}

func (l *hookLog) add(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

func (l *hookLog) take() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.events
	l.events = nil
	return out
}

func recordingMiddleware(log *hookLog, label string) MiddlewareFunc {
	return MiddlewareFunc{
		BeforeFn: func(ctx context.Context, tc *ToolContext) (context.Context, error) {
			log.add(label + ".before:" + string(tc.Name))
			return ctx, nil
		},
		AfterFn: func(_ context.Context, tc *ToolContext, result *mcp.CallToolResult, _ error) (*mcp.CallToolResult, error) {
			log.add(label + ".after:" + string(tc.Name))
			return result, nil
		},
	}
}

// callHarness is a toolkit with one hook of every kind, each recording into
// log, over a mock client that records the SQL it was sent.
type callHarness struct {
	toolkit *Toolkit
	log     *hookLog
	sqlMu   sync.Mutex
	sql     []string
}

const transformMarker = "[transformed]"

func newCallHarness(t *testing.T) *callHarness {
	t.Helper()
	h := &callHarness{log: &hookLog{}}

	mock := NewMockTrinoClient()
	baseQuery := mock.QueryFunc
	mock.QueryFunc = func(ctx context.Context, sql string, opts client.QueryOptions) (*client.QueryResult, error) {
		h.sqlMu.Lock()
		h.sql = append(h.sql, sql)
		h.sqlMu.Unlock()
		return baseQuery(ctx, sql, opts)
	}
	baseExplain := mock.ExplainFunc
	mock.ExplainFunc = func(ctx context.Context, sql string, et client.ExplainType) (*client.ExplainResult, error) {
		h.sqlMu.Lock()
		h.sql = append(h.sql, sql)
		h.sqlMu.Unlock()
		return baseExplain(ctx, sql, et)
	}

	opts := make([]ToolkitOption, 0, len(AllTools())+3)
	opts = append(opts, WithMiddleware(recordingMiddleware(h.log, "global")))
	for _, name := range AllTools() {
		opts = append(opts, WithToolMiddleware(name, recordingMiddleware(h.log, "tool")))
	}
	opts = append(opts,
		WithQueryInterceptor(QueryInterceptorFunc(func(_ context.Context, sql string, name ToolName) (string, error) {
			h.log.add("intercept:" + string(name))
			return sql + " /* intercepted */", nil
		})),
		WithResultTransformer(ResultTransformerFunc(func(_ context.Context, name ToolName, r *mcp.CallToolResult) (*mcp.CallToolResult, error) {
			h.log.add("transform:" + string(name))
			r.Content = append(r.Content, &mcp.TextContent{Text: transformMarker})
			return r, nil
		})),
	)
	h.toolkit = NewToolkit(mock, DefaultConfig(), opts...)
	return h
}

func (h *callHarness) takeSQL() []string {
	h.sqlMu.Lock()
	defer h.sqlMu.Unlock()
	out := h.sql
	h.sql = nil
	return out
}

// session registers every tool on a server, each with a per-registration
// middleware, and returns a client session connected to it in memory.
func (h *callHarness) session(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	for _, name := range AllTools() {
		h.toolkit.RegisterWith(server, name, WithPerToolMiddleware(recordingMiddleware(h.log, "reg")))
	}

	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "1.0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func texts(r *mcp.CallToolResult) []string {
	var out []string
	for _, c := range r.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out = append(out, tc.Text)
		}
	}
	return out
}

// asJSON normalizes v to its decoded JSON form so a value from the wire and
// one returned in-process compare equal when they encode the same.
func asJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestCall_MatchesRegisteredTool(t *testing.T) {
	sqlTool := func(name ToolName) []string {
		return []string{
			"global.before:" + string(name), "tool.before:" + string(name), "reg.before:" + string(name),
			"intercept:" + string(name),
			"reg.after:" + string(name), "tool.after:" + string(name), "global.after:" + string(name),
			"transform:" + string(name),
		}
	}
	metaTool := func(name ToolName) []string {
		return []string{
			"global.before:" + string(name), "tool.before:" + string(name), "reg.before:" + string(name),
			"reg.after:" + string(name), "tool.after:" + string(name), "global.after:" + string(name),
			"transform:" + string(name),
		}
	}

	tests := []struct {
		name       ToolName
		args       map[string]any
		input      any
		wantEvents []string
		wantSQL    []string
	}{
		{
			name:       ToolQuery,
			args:       map[string]any{"sql": "SELECT 1"},
			input:      QueryInput{SQL: "SELECT 1"},
			wantEvents: sqlTool(ToolQuery),
			wantSQL:    []string{"SELECT 1 /* intercepted */"},
		},
		{
			name:       ToolExecute,
			args:       map[string]any{"sql": "INSERT INTO t VALUES (1)"},
			input:      ExecuteInput{SQL: "INSERT INTO t VALUES (1)"},
			wantEvents: sqlTool(ToolExecute),
			wantSQL:    []string{"INSERT INTO t VALUES (1) /* intercepted */"},
		},
		{
			name:       ToolExplain,
			args:       map[string]any{"sql": "SELECT 1"},
			input:      ExplainInput{SQL: "SELECT 1"},
			wantEvents: sqlTool(ToolExplain),
			wantSQL:    []string{"SELECT 1 /* intercepted */"},
		},
		{
			name:       ToolBrowse,
			args:       map[string]any{"catalog": "memory"},
			input:      BrowseInput{Catalog: "memory"},
			wantEvents: metaTool(ToolBrowse),
		},
		{
			name:       ToolDescribeTable,
			args:       map[string]any{"catalog": "memory", "schema": "default", "table": "users"},
			input:      DescribeTableInput{Catalog: "memory", Schema: "default", Table: "users"},
			wantEvents: metaTool(ToolDescribeTable),
		},
		{
			name:       ToolListConnections,
			args:       map[string]any{},
			input:      ListConnectionsInput{},
			wantEvents: metaTool(ToolListConnections),
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			ctx := context.Background()
			h := newCallHarness(t)
			cs := h.session(t)

			registered, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: string(tt.name), Arguments: tt.args})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			regEvents, regSQL := h.log.take(), h.takeSQL()

			result, out, err := h.toolkit.Call(ctx, &mcp.CallToolRequest{}, tt.name, tt.input,
				WithPerToolMiddleware(recordingMiddleware(h.log, "reg")))
			if err != nil {
				t.Fatalf("Call: %v", err)
			}
			callEvents, callSQL := h.log.take(), h.takeSQL()

			if registered.IsError || result.IsError {
				t.Fatalf("unexpected error result: registered=%v call=%v", texts(registered), texts(result))
			}
			if !reflect.DeepEqual(regEvents, tt.wantEvents) {
				t.Errorf("registered hooks = %v, want %v", regEvents, tt.wantEvents)
			}
			if !reflect.DeepEqual(callEvents, tt.wantEvents) {
				t.Errorf("Call hooks = %v, want %v", callEvents, tt.wantEvents)
			}
			if !reflect.DeepEqual(regSQL, tt.wantSQL) || !reflect.DeepEqual(callSQL, tt.wantSQL) {
				t.Errorf("SQL sent: registered=%q call=%q, want %q", regSQL, callSQL, tt.wantSQL)
			}

			gotTexts := texts(result)
			if !reflect.DeepEqual(gotTexts, texts(registered)) {
				t.Errorf("content differs:\n Call:       %q\n registered: %q", gotTexts, texts(registered))
			}
			if len(gotTexts) == 0 || gotTexts[len(gotTexts)-1] != transformMarker {
				t.Errorf("Call content %q does not end with the transformer's marker", gotTexts)
			}
			if out == nil {
				t.Fatal("Call returned no structured output")
			}
			if got, want := asJSON(t, out), asJSON(t, registered.StructuredContent); !reflect.DeepEqual(got, want) {
				t.Errorf("structured output differs:\n Call:       %v\n registered: %v", got, want)
			}
		})
	}
}

func TestCall_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		tool    ToolName
		input   any
		wantMsg string
	}{
		{name: "unknown tool", tool: ToolName("trino_nope"), input: QueryInput{SQL: "SELECT 1"}, wantMsg: `unknown tool: "trino_nope"`},
		{
			name: "wrong input type", tool: ToolQuery, input: ExecuteInput{SQL: "SELECT 1"},
			wantMsg: "input must be tools.QueryInput, got tools.ExecuteInput",
		},
		{
			name: "pointer input", tool: ToolQuery, input: &QueryInput{SQL: "SELECT 1"},
			wantMsg: "input must be tools.QueryInput, got *tools.QueryInput",
		},
		{name: "nil input", tool: ToolBrowse, input: nil, wantMsg: "input must be tools.BrowseInput, got <nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newCallHarness(t)
			result, out, err := h.toolkit.Call(context.Background(), &mcp.CallToolRequest{}, tt.tool, tt.input)
			if err != nil {
				t.Fatalf("Call returned a Go error %v; want an error result", err)
			}
			if result == nil || !result.IsError {
				t.Fatalf("result = %+v, want an error result", result)
			}
			if out != nil {
				t.Errorf("out = %v, want nil", out)
			}
			if msg := strings.Join(texts(result), "\n"); !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("message %q does not contain %q", msg, tt.wantMsg)
			}
			if events := h.log.take(); len(events) != 0 {
				t.Errorf("hooks ran for a rejected call: %v", events)
			}
			if sql := h.takeSQL(); len(sql) != 0 {
				t.Errorf("SQL reached the client for a rejected call: %q", sql)
			}
		})
	}
}

func TestCall_MiddlewareRejectionStopsHandler(t *testing.T) {
	h := newCallHarness(t)
	deny := BeforeFunc(func(ctx context.Context, _ *ToolContext) (context.Context, error) {
		return ctx, errors.New("denied")
	})
	result, _, err := h.toolkit.Call(context.Background(), &mcp.CallToolRequest{}, ToolQuery,
		QueryInput{SQL: "SELECT 1"}, WithPerToolMiddleware(deny))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !result.IsError || !strings.Contains(strings.Join(texts(result), ""), "middleware error: denied") {
		t.Errorf("result = %q, want middleware error result", texts(result))
	}
	if sql := h.takeSQL(); len(sql) != 0 {
		t.Errorf("SQL reached the client after the per-call middleware denied it: %q", sql)
	}
}

func TestHandlerFor_UnknownNamePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("handlerFor did not panic on an unknown tool name")
		}
	}()
	NewToolkit(NewMockTrinoClient(), DefaultConfig()).handlerFor(ToolName("trino_nope"), nil)
}
