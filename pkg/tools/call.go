package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handlerFunc is the untyped handler shape the middleware chain wraps.
type handlerFunc func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error)

// builtinTool is a built-in tool's handler and the input type it accepts.
type builtinTool struct {
	run        handlerFunc
	checkInput func(input any) error
}

// adapt turns a typed tool handler into a builtinTool whose run rejects any
// input other than T with an error result.
func adapt[T any](h func(context.Context, *mcp.CallToolRequest, T) (*mcp.CallToolResult, any, error)) builtinTool {
	check := func(input any) error {
		if _, ok := input.(T); !ok {
			var want T
			return fmt.Errorf("input must be %T, got %T", want, input)
		}
		return nil
	}
	return builtinTool{
		run: func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
			typed, ok := input.(T)
			if !ok {
				return ErrorResult("internal error: invalid input type"), nil, nil
			}
			return h(ctx, req, typed)
		},
		checkInput: check,
	}
}

// builtin returns the built-in tool named name, or false for an unknown name.
func (t *Toolkit) builtin(name ToolName) (builtinTool, bool) {
	switch name {
	case ToolQuery:
		return adapt(t.handleQuery), true
	case ToolExecute:
		return adapt(t.handleExecute), true
	case ToolExplain:
		return adapt(t.handleExplain), true
	case ToolBrowse:
		return adapt(t.handleBrowse), true
	case ToolDescribeTable:
		return adapt(t.handleDescribeTable), true
	case ToolListConnections:
		return adapt(func(ctx context.Context, req *mcp.CallToolRequest, _ ListConnectionsInput) (*mcp.CallToolResult, any, error) {
			return t.handleListConnections(ctx, req)
		}), true
	default:
		return builtinTool{}, false
	}
}

// handlerFor returns the built-in tool's handler wrapped with the toolkit's
// middleware and transformers and cfg's per-registration middleware. Both
// registration and Call build their handler here, so the two run the same
// chain. It panics on an unknown name; callers check the name first.
func (t *Toolkit) handlerFor(name ToolName, cfg *toolConfig) handlerFunc {
	b, ok := t.builtin(name)
	if !ok {
		panic(fmt.Sprintf("tools: no built-in tool %q", name))
	}
	return t.wrapHandler(name, b.run, cfg)
}

// newToolConfig applies per-registration options.
func newToolConfig(opts []ToolOption) *toolConfig {
	cfg := &toolConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Call runs the named built-in tool's handler as a registered call would,
// without registering the tool on an mcp.Server: global, per-tool and
// per-registration (opts) middleware Before and After hooks, the query
// interceptors the handler applies, and the result transformers.
//
// It lets a caller publish a tool under its own name, or route calls for
// several engines through one tool, and still use this toolkit's Trino
// handling. ToolContext.Name and the name passed to interceptors and
// transformers stay the built-in ToolName, so hooks keyed on ToolQuery and the
// like behave the same whether the call arrived through Register or Call.
//
// The input must be the tool's input type, passed by value: QueryInput,
// ExecuteInput, ExplainInput, BrowseInput, DescribeTableInput or
// ListConnectionsInput. An unknown name or any other input type returns an
// error result before any middleware runs. Options that only shape the
// advertised tool (WithTitle, WithDescription, and so on) have no effect.
//
// The second return value is the handler's structured output (for example
// *QueryOutput), or nil where the handler returned none. Call does what the
// SDK does not do for an unregistered handler, and nothing more: it does not
// validate input against the tool's input schema or apply schema defaults, and
// it does not marshal the output into result.StructuredContent. A caller that
// returns the output from its own typed mcp.AddTool handler gets that from the
// SDK as usual.
func (t *Toolkit) Call(
	ctx context.Context, req *mcp.CallToolRequest, name ToolName, input any, opts ...ToolOption,
) (*mcp.CallToolResult, any, error) {
	b, ok := t.builtin(name)
	if !ok {
		return ErrorResult(fmt.Sprintf("unknown tool: %q", name)), nil, nil
	}
	if err := b.checkInput(input); err != nil {
		return ErrorResult(fmt.Sprintf("%s: %v", name, err)), nil, nil
	}
	return t.handlerFor(name, newToolConfig(opts))(ctx, req, input)
}
