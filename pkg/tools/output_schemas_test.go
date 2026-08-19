package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDefaultOutputSchema(t *testing.T) {
	t.Run("returns schema for known tool", func(t *testing.T) {
		if schema := DefaultOutputSchema(ToolQuery); schema == nil {
			t.Error("expected non-nil schema for ToolQuery")
		}
	})

	t.Run("returns nil for unknown tool", func(t *testing.T) {
		if schema := DefaultOutputSchema("unknown_tool"); schema != nil {
			t.Errorf("expected nil schema for unknown tool, got %v", schema)
		}
	})

	t.Run("all tools have defaults", func(t *testing.T) {
		for _, name := range AllTools() {
			if DefaultOutputSchema(name) == nil {
				t.Errorf("tool %s has no default output schema", name)
			}
		}
	})

	t.Run("schema is map[string]any with type object", func(t *testing.T) {
		for _, name := range AllTools() {
			m, ok := DefaultOutputSchema(name).(map[string]any)
			if !ok {
				t.Errorf("tool %s schema is not map[string]any", name)
				continue
			}
			if m["type"] != "object" {
				t.Errorf("tool %s schema type = %v, want %q", name, m["type"], "object")
			}
			if _, ok := m["properties"].(map[string]any); !ok {
				t.Errorf("tool %s schema has no properties object", name)
			}
		}
	})

	t.Run("query and execute schemas do not alias", func(t *testing.T) {
		q, ok := defaultOutputSchemas[ToolQuery].(map[string]any)
		if !ok {
			t.Fatal("trino_query schema is not map[string]any")
		}
		e, ok := defaultOutputSchemas[ToolExecute].(map[string]any)
		if !ok {
			t.Fatal("trino_execute schema is not map[string]any")
		}
		if fmt.Sprintf("%p", q) == fmt.Sprintf("%p", e) {
			t.Error("trino_query and trino_execute share one schema map; mutating one would mutate both")
		}
	})

	t.Run("returns a deep copy the caller cannot mutate through", func(t *testing.T) {
		schema, ok := DefaultOutputSchema(ToolQuery).(map[string]any)
		if !ok {
			t.Fatal("trino_query schema is not map[string]any")
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("schema has no properties")
		}
		props["injected_by_caller"] = map[string]any{"type": "string"}

		rows, ok := props["rows"].(map[string]any)
		if !ok {
			t.Fatal("schema has no rows property")
		}
		rows["type"] = "array" // the null-rejecting shape, written through the copy

		fresh, ok := DefaultOutputSchema(ToolQuery).(map[string]any)
		if !ok {
			t.Fatal("trino_query schema is not map[string]any")
		}
		freshProps, ok := fresh["properties"].(map[string]any)
		if !ok {
			t.Fatal("fresh schema has no properties")
		}
		if _, leaked := freshProps["injected_by_caller"]; leaked {
			t.Error("mutating the returned schema changed the package default")
		}
		freshRows, ok := freshProps["rows"].(map[string]any)
		if !ok {
			t.Fatal("fresh schema has no rows property")
		}
		if _, ok := freshRows["type"].([]string); !ok {
			t.Errorf("rows type = %#v, want the untouched []string union", freshRows["type"])
		}
	})
}

// TestDefaultOutputSchema_Open pins the end state of issue #85: every advertised
// schema is open at the top level, so a host that adds keys to structuredContent
// (an error envelope, a call reference) does not invalidate the result.
func TestDefaultOutputSchema_Open(t *testing.T) {
	for _, name := range AllTools() {
		t.Run(string(name), func(t *testing.T) {
			m, ok := DefaultOutputSchema(name).(map[string]any)
			if !ok {
				t.Fatalf("tool %s schema is not map[string]any", name)
			}
			if ap, ok := m["additionalProperties"]; ok && ap == false {
				t.Error(`schema declares "additionalProperties": false; it must stay open`)
			}
			if req, ok := m["required"]; ok {
				t.Errorf(`schema declares "required": %v; it must not constrain required keys`, req)
			}
		})
	}
}

// TestRegisteredOutputSchemas_Open asserts the same openness on what the server
// actually advertises over the wire, not just on the map literals.
func TestRegisteredOutputSchemas_Open(t *testing.T) {
	ctx := context.Background()
	tools := listServerTools(ctx, t)

	for _, name := range AllTools() {
		tool, ok := tools[string(name)]
		if !ok {
			t.Errorf("tool %s was not advertised", name)
			continue
		}
		t.Run(string(name), func(t *testing.T) {
			if tool.OutputSchema == nil {
				t.Fatal("tool advertises no output schema")
			}
			var m map[string]any
			raw, err := json.Marshal(tool.OutputSchema)
			if err != nil {
				t.Fatalf("marshal output schema: %v", err)
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("unmarshal output schema: %v", err)
			}
			if m["type"] != "object" {
				t.Errorf("advertised schema type = %v, want %q", m["type"], "object")
			}
			if ap, ok := m["additionalProperties"]; ok && ap == false {
				t.Error(`advertised schema declares "additionalProperties": false`)
			}
			if req, ok := m["required"]; ok {
				t.Errorf(`advertised schema declares "required": %v`, req)
			}
		})
	}
}

// TestDefaultOutputSchema_NullableArrays guards against a regression that only
// shows up on error paths: a nil Go slice marshals as null, and the SDK
// validates structured output even when a handler returns an error result with
// no typed output. An array property that rejects null turns every such call
// into a protocol error that discards the tool's own message.
func TestDefaultOutputSchema_NullableArrays(t *testing.T) {
	nilable := map[ToolName][]string{
		ToolQuery:           {"columns", "rows"},
		ToolExecute:         {"columns", "rows"},
		ToolBrowse:          {"items"},
		ToolDescribeTable:   {"columns", "sample"},
		ToolListConnections: {"connections"},
	}

	for name, props := range nilable {
		for _, prop := range props {
			t.Run(fmt.Sprintf("%s/%s", name, prop), func(t *testing.T) {
				schema, ok := DefaultOutputSchema(name).(map[string]any)
				if !ok {
					t.Fatalf("tool %s has no map schema", name)
				}
				properties, ok := schema["properties"].(map[string]any)
				if !ok {
					t.Fatalf("tool %s schema has no properties", name)
				}
				propSchema, ok := properties[prop].(map[string]any)
				if !ok {
					t.Fatalf("tool %s has no schema for property %q", name, prop)
				}
				types, ok := propSchema["type"].([]string)
				if !ok {
					t.Fatalf("property %q type = %#v, want a list of types including \"null\"", prop, propSchema["type"])
				}
				if !slices.Contains(types, "null") {
					t.Errorf("property %q type = %v, want it to include \"null\"", prop, types)
				}
			})
		}
	}
}

// TestErrorResultPassesOutputValidation exercises the nullable-array reasoning
// end to end: a handler that fails must surface its own error message, not a
// JSON-RPC output-validation error.
func TestErrorResultPassesOutputValidation(t *testing.T) {
	ctx := context.Background()
	mock := NewMockTrinoClient()
	toolkit := NewToolkit(mock, DefaultConfig())
	session := connectToolkit(ctx, t, toolkit)

	// An empty sql argument makes every SQL handler return ErrorResult with no
	// typed output, which is the case that trips output validation.
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      string(ToolQuery),
		Arguments: map[string]any{"sql": ""},
	})
	if err != nil {
		t.Fatalf("CallTool returned a protocol error instead of a tool error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected an error result for empty sql")
	}
}

// TestSuccessfulResultsValidateAgainstSchema calls every tool through MCP
// dispatch and asserts the real output validates against the advertised schema.
// The SDK turns a validation failure into a protocol error, so a schema that is
// narrower than what a handler emits breaks the tool outright.
func TestSuccessfulResultsValidateAgainstSchema(t *testing.T) {
	ctx := context.Background()
	session := connectToolkit(ctx, t, NewToolkit(NewMockTrinoClient(), DefaultConfig()))

	calls := []struct {
		tool ToolName
		args map[string]any
	}{
		{ToolQuery, map[string]any{"sql": "SELECT 1"}},
		{ToolExecute, map[string]any{"sql": "INSERT INTO t VALUES (1)"}},
		{ToolExplain, map[string]any{"sql": "SELECT 1"}},
		{ToolBrowse, map[string]any{}},
		{ToolBrowse, map[string]any{"catalog": "hive"}},
		{ToolBrowse, map[string]any{"catalog": "hive", "schema": "default"}},
		{ToolDescribeTable, map[string]any{"catalog": "hive", "schema": "default", "table": "users"}},
		{ToolDescribeTable, map[string]any{
			"catalog": "hive", "schema": "default", "table": "users", "include_sample": true,
		}},
		{ToolListConnections, map[string]any{}},
	}

	for _, call := range calls {
		t.Run(fmt.Sprintf("%s/%v", call.tool, call.args), func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      string(call.tool),
				Arguments: call.args,
			})
			if err != nil {
				t.Fatalf("CallTool returned a protocol error: %v", err)
			}
			if result.IsError {
				t.Fatalf("tool returned an error result: %v", result.Content)
			}
			if result.StructuredContent == nil {
				t.Error("tool returned no structured content")
			}
		})
	}
}

func TestGetOutputSchema(t *testing.T) {
	// Sentinel strings stand in for schemas so the priority chain can be
	// compared without comparing non-comparable maps.
	const sentinelToolkit = "toolkit-schema"
	const sentinelReg = "registration-schema"

	t.Run("returns default when no overrides", func(t *testing.T) {
		tk := &Toolkit{}
		if schema := tk.getOutputSchema(ToolQuery, nil); schema == nil {
			t.Error("expected non-nil default output schema")
		}
	})

	t.Run("toolkit-level override wins over default", func(t *testing.T) {
		tk := &Toolkit{outputSchemas: map[ToolName]any{ToolQuery: sentinelToolkit}}
		if schema := tk.getOutputSchema(ToolQuery, nil); schema != sentinelToolkit {
			t.Errorf("expected toolkit override, got %v", schema)
		}
	})

	t.Run("per-registration override wins over toolkit", func(t *testing.T) {
		tk := &Toolkit{outputSchemas: map[ToolName]any{ToolQuery: sentinelToolkit}}
		cfg := &toolConfig{outputSchema: sentinelReg}
		if schema := tk.getOutputSchema(ToolQuery, cfg); schema != sentinelReg {
			t.Errorf("expected per-registration override, got %v", schema)
		}
	})

	t.Run("nil config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{outputSchemas: map[ToolName]any{ToolBrowse: sentinelToolkit}}
		if schema := tk.getOutputSchema(ToolBrowse, nil); schema != sentinelToolkit {
			t.Errorf("expected toolkit override, got %v", schema)
		}
	})

	t.Run("nil outputSchema in config falls through to toolkit", func(t *testing.T) {
		tk := &Toolkit{outputSchemas: map[ToolName]any{ToolQuery: sentinelToolkit}}
		cfg := &toolConfig{outputSchema: nil}
		if schema := tk.getOutputSchema(ToolQuery, cfg); schema != sentinelToolkit {
			t.Errorf("expected toolkit override when cfg.outputSchema is nil, got %v", schema)
		}
	})

	t.Run("empty toolkit map falls through to default", func(t *testing.T) {
		tk := &Toolkit{outputSchemas: map[ToolName]any{}}
		m, ok := tk.getOutputSchema(ToolExplain, nil).(map[string]any)
		if !ok {
			t.Fatal("expected the default schema as map[string]any")
		}
		if m["type"] != "object" {
			t.Errorf("expected schema type 'object', got %v", m["type"])
		}
	})
}

func TestWithOutputSchemas(t *testing.T) {
	custom := map[string]any{"type": "object", "properties": map[string]any{"plan": map[string]any{"type": "string"}}}
	tk := NewToolkit(NewMockTrinoClient(), DefaultConfig(),
		WithOutputSchemas(map[ToolName]any{ToolExplain: custom}),
	)

	schema, ok := tk.getOutputSchema(ToolExplain, nil).(map[string]any)
	if !ok {
		t.Fatal("expected the override as map[string]any")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("override schema has no properties")
	}
	if _, ok := props["plan"]; !ok {
		t.Error("expected the override schema, got the default")
	}

	// Untouched tools keep their defaults.
	if tk.getOutputSchema(ToolQuery, nil) == nil {
		t.Error("expected trino_query to keep its default schema")
	}
}

// TestWithOutputSchema_Advertised verifies a per-registration override reaches
// the wire, not just the resolution chain.
func TestWithOutputSchema_Advertised(t *testing.T) {
	ctx := context.Background()
	custom := map[string]any{
		"type":       "object",
		"properties": map[string]any{"plan": map[string]any{"type": "string"}},
	}

	toolkit := NewToolkit(NewMockTrinoClient(), DefaultConfig())
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	toolkit.RegisterWith(server, ToolExplain, WithOutputSchema(custom))

	tool, ok := listTools(ctx, t, server)[string(ToolExplain)]
	if !ok {
		t.Fatal("trino_explain was not advertised")
	}
	raw, err := json.Marshal(tool.OutputSchema)
	if err != nil {
		t.Fatalf("marshal output schema: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal output schema: %v", err)
	}
	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatal("advertised schema has no properties")
	}
	if len(props) != 1 {
		t.Errorf("advertised %d properties, want only the overridden one: %v", len(props), props)
	}
}

// TestDeepCopySchema covers the shapes a schema literal can hold, including the
// []any branch that no default schema currently uses — an override supplied by a
// composing host can, and a shallow copy there would let the caller reach back
// into the package default.
func TestDeepCopySchema(t *testing.T) {
	original := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"rows":  map[string]any{"type": []string{"array", "null"}},
			"enum":  []any{"a", map[string]any{"nested": "value"}},
			"count": 3,
		},
	}

	copied, ok := deepCopySchema(original).(map[string]any)
	if !ok {
		t.Fatal("deepCopySchema did not return map[string]any")
	}
	if !reflect.DeepEqual(original, copied) {
		t.Fatalf("copy differs from original:\n got %#v\nwant %#v", copied, original)
	}

	// Mutate every branch through the copy; none may reach the original.
	copiedProps, ok := copied["properties"].(map[string]any)
	if !ok {
		t.Fatal("copy has no properties map")
	}
	copiedProps["injected"] = true

	copiedRows, ok := copiedProps["rows"].(map[string]any)
	if !ok {
		t.Fatal("copy has no rows map")
	}
	copiedRowsType, ok := copiedRows["type"].([]string)
	if !ok {
		t.Fatal("copy has no rows type union")
	}
	copiedRowsType[0] = "mutated"

	copiedEnum, ok := copiedProps["enum"].([]any)
	if !ok {
		t.Fatal("copy has no enum slice")
	}
	copiedEnum[0] = "mutated"
	copiedNested, ok := copiedEnum[1].(map[string]any)
	if !ok {
		t.Fatal("copy has no nested enum map")
	}
	copiedNested["nested"] = "mutated"

	props, ok := original["properties"].(map[string]any)
	if !ok {
		t.Fatal("original lost its properties map")
	}
	if _, leaked := props["injected"]; leaked {
		t.Error("adding a key to the copy changed the original map")
	}
	rows, ok := props["rows"].(map[string]any)
	if !ok {
		t.Fatal("original lost its rows map")
	}
	rowsType, ok := rows["type"].([]string)
	if !ok {
		t.Fatal("original lost its rows type union")
	}
	if rowsType[0] != "array" {
		t.Errorf("original rows type[0] = %q, want %q — []string was shared", rowsType[0], "array")
	}
	enum, ok := props["enum"].([]any)
	if !ok {
		t.Fatal("original lost its enum slice")
	}
	if enum[0] != "a" {
		t.Errorf("original enum[0] = %v, want \"a\" — []any was shared", enum[0])
	}
	nested, ok := enum[1].(map[string]any)
	if !ok {
		t.Fatal("original lost its nested enum map")
	}
	if got := nested["nested"]; got != "value" {
		t.Errorf("original enum[1].nested = %v, want \"value\" — nested map was shared", got)
	}

	// A scalar and an unhandled type pass through unchanged.
	if got := deepCopySchema(42); got != 42 {
		t.Errorf("deepCopySchema(42) = %v, want 42", got)
	}
	if got := deepCopySchema(nil); got != nil {
		t.Errorf("deepCopySchema(nil) = %v, want nil", got)
	}
}

// TestDocumentedOverrideExampleSurvivesErrorPath runs the schema written in the
// WithOutputSchemas godoc and in docs/library/extensibility.md through a failing
// call. An earlier draft of that example declared "rows" as a bare "array",
// which rejects the null a nil slice marshals to and turned every failed call
// into a protocol error. A doc example is copy-pasted verbatim, so it is worth
// a test.
func TestDocumentedOverrideExampleSurvivesErrorPath(t *testing.T) {
	ctx := context.Background()
	documented := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"rows": map[string]any{"type": []string{"array", "null"}},
		},
	}

	toolkit := NewToolkit(NewMockTrinoClient(), DefaultConfig(),
		WithOutputSchemas(map[ToolName]any{ToolQuery: documented}),
	)
	session := connectToolkit(ctx, t, toolkit)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      string(ToolQuery),
		Arguments: map[string]any{"sql": ""},
	})
	if err != nil {
		t.Fatalf("the documented override turned a tool error into a protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected an error result for empty sql")
	}
}

// TestOutputSchemasCoverEveryOutputField walks the Go output structs and fails
// when a json tag has no declared property. The schemas are open, so a missing
// property still validates — it just silently stops being documented. Nothing
// else ties the two together, so without this they drift.
func TestOutputSchemasCoverEveryOutputField(t *testing.T) {
	cases := []struct {
		tool ToolName
		out  any
	}{
		{ToolQuery, QueryOutput{}},
		{ToolExecute, QueryOutput{}},
		{ToolExplain, ExplainOutput{}},
		{ToolBrowse, BrowseOutput{}},
		{ToolDescribeTable, DescribeTableOutput{}},
		{ToolListConnections, ListConnectionsOutput{}},
	}

	for _, tc := range cases {
		t.Run(string(tc.tool), func(t *testing.T) {
			schema, ok := DefaultOutputSchema(tc.tool).(map[string]any)
			if !ok {
				t.Fatalf("tool %s has no map schema", tc.tool)
			}
			assertPropertiesCoverStruct(t, schema, reflect.TypeOf(tc.out), string(tc.tool))
		})
	}
}

// assertPropertiesCoverStruct checks that every json-tagged field of rt has a
// property in schema, recursing into struct fields and slices of structs.
func assertPropertiesCoverStruct(t *testing.T, schema map[string]any, rt reflect.Type, path string) {
	t.Helper()

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s: schema has no properties object", path)
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}

		prop, ok := props[name].(map[string]any)
		if !ok {
			t.Errorf("%s: field %q (%s) has no declared property", path, name, field.Type)
			continue
		}

		switch ft := field.Type; ft.Kind() {
		case reflect.Struct:
			assertPropertiesCoverStruct(t, prop, ft, path+"."+name)
		case reflect.Slice:
			if elem := ft.Elem(); elem.Kind() == reflect.Struct {
				items, ok := prop["items"].(map[string]any)
				if !ok {
					t.Errorf("%s.%s: array property has no items schema", path, name)
					continue
				}
				assertPropertiesCoverStruct(t, items, elem, path+"."+name+"[]")
			}
		default:
		}
	}
}

// listServerTools registers every tool on a fresh server and returns what a
// client sees, keyed by tool name.
func listServerTools(ctx context.Context, t *testing.T) map[string]*mcp.Tool {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	NewToolkit(NewMockTrinoClient(), DefaultConfig()).RegisterAll(server)
	return listTools(ctx, t, server)
}

// listTools connects a client to server over an in-memory transport and returns
// the advertised tools keyed by name.
func listTools(ctx context.Context, t *testing.T, server *mcp.Server) map[string]*mcp.Tool {
	t.Helper()

	session := connectSession(ctx, t, server)
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	advertised := make(map[string]*mcp.Tool, len(result.Tools))
	for _, tool := range result.Tools {
		advertised[tool.Name] = tool
	}
	return advertised
}

// connectToolkit registers every tool from toolkit and returns a client session.
func connectToolkit(ctx context.Context, t *testing.T, toolkit *Toolkit) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	toolkit.RegisterAll(server)
	return connectSession(ctx, t, server)
}

// connectSession wires a client to server over an in-memory transport, closing
// both sessions when the test ends.
func connectSession(ctx context.Context, t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}
