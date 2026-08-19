package tools

// nullableArray returns an array schema that also admits JSON null.
//
// A nil Go slice marshals as null, and null is not an "array". That matters
// beyond empty results: the SDK's typed-handler wrapper validates output even
// for error results, substituting the zero value of the output struct whenever
// a handler returns an error result with no typed output. Rejecting null there
// turns every failed call into a JSON-RPC output-validation error and discards
// the tool's own error message.
func nullableArray(items any) map[string]any {
	return map[string]any{
		"type":  []string{"array", "null"},
		"items": items,
	}
}

// defaultOutputSchemas holds the default JSON Schema (2020-12) for each tool's
// structured output. Schemas use map[string]any so they can be remarshaled by
// the MCP SDK's schema resolution pipeline.
//
// All schemas declare only "type", "properties" and "items" — no "required"
// constraints and no "additionalProperties": false — so that partial results
// and implementation-specific fields never fail schema validation at runtime.
// A host that composes this toolkit and adds keys to structuredContent (an
// error envelope, a call reference) stays valid against what the tool
// advertises. This matches the posture of mcp-s3 and mcp-datahub, rather than
// the closed schema jsonschema-go infers from the Go output structs.
//
// Properties backed by a Go slice are declared with nullableArray rather than a
// bare "array" type. See that function for why null must be admitted.
var defaultOutputSchemas = map[ToolName]any{
	ToolQuery:   queryOutputSchema(),
	ToolExecute: queryOutputSchema(),

	ToolExplain: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plan": map[string]any{"type": "string"},
			"type": map[string]any{"type": "string"},
		},
	},

	ToolBrowse: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"level":   map[string]any{"type": "string"},
			"catalog": map[string]any{"type": "string"},
			"schema":  map[string]any{"type": "string"},
			"items":   nullableArray(map[string]any{"type": "string"}),
			"count":   map[string]any{"type": "integer"},
			"pattern": map[string]any{"type": "string"},
		},
	},

	ToolDescribeTable: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"catalog": map[string]any{"type": "string"},
			"schema":  map[string]any{"type": "string"},
			"table":   map[string]any{"type": "string"},
			"columns": nullableArray(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":     map[string]any{"type": "string"},
					"type":     map[string]any{"type": "string"},
					"nullable": map[string]any{"type": "string"},
					"comment":  map[string]any{"type": "string"},
				},
			}),
			"column_count": map[string]any{"type": "integer"},
			"sample":       nullableArray(rowSchema()),
		},
	},

	ToolListConnections: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connections": nullableArray(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":       map[string]any{"type": "string"},
					"host":       map[string]any{"type": "string"},
					"port":       map[string]any{"type": "integer"},
					"catalog":    map[string]any{"type": "string"},
					"schema":     map[string]any{"type": "string"},
					"ssl":        map[string]any{"type": "boolean"},
					"is_default": map[string]any{"type": "boolean"},
				},
			}),
			"count": map[string]any{"type": "integer"},
		},
	},
}

// queryOutputSchema returns the schema for QueryOutput, shared by trino_query
// and trino_execute. Each call builds a fresh map so the two entries never
// alias one another.
func queryOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"columns": nullableArray(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"type": map[string]any{"type": "string"},
				},
			}),
			"rows":      nullableArray(rowSchema()),
			"row_count": map[string]any{"type": "integer"},
			"stats": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"row_count":     map[string]any{"type": "integer"},
					"truncated":     map[string]any{"type": "boolean"},
					"limit_applied": map[string]any{"type": "integer"},
					"duration_ms":   map[string]any{"type": "integer"},
				},
			},
		},
	}
}

// rowSchema returns the schema for one result row: a map of column name to an
// unconstrained value, so any Trino type round-trips.
func rowSchema() map[string]any {
	return map[string]any{
		"type":                 []string{"object", "null"},
		"additionalProperties": true,
	}
}

// DefaultOutputSchema returns the default JSON Schema for a tool's structured
// output. Returns nil for unknown tool names.
//
// The result is a deep copy, so a caller that adapts a default — adding a
// property, widening a type — cannot reach through it and change what every
// later Toolkit advertises. Downstream consumers otherwise have to copy
// defensively before touching it.
func DefaultOutputSchema(name ToolName) any {
	schema, ok := defaultOutputSchemas[name]
	if !ok {
		return nil
	}
	return deepCopySchema(schema)
}

// deepCopySchema copies the map, slice and scalar values a schema literal is
// built from. Values of any other type are returned as-is; the schemas here
// contain none, and a shared scalar is harmless.
func deepCopySchema(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, elem := range val {
			out[k] = deepCopySchema(elem)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = deepCopySchema(elem)
		}
		return out
	case []string:
		return append([]string(nil), val...)
	default:
		return v
	}
}

// getOutputSchema resolves the output schema for a tool using the priority chain:
// 1. Per-registration override (cfg.outputSchema) — highest priority
// 2. Toolkit-level override (t.outputSchemas) — medium priority
// 3. Default output schema — lowest priority.
func (t *Toolkit) getOutputSchema(name ToolName, cfg *toolConfig) any {
	// Per-registration override (highest priority)
	if cfg != nil && cfg.outputSchema != nil {
		return cfg.outputSchema
	}

	// Toolkit-level override (medium priority)
	if schema, ok := t.outputSchemas[name]; ok {
		return schema
	}

	// Default output schema (lowest priority). Copied for the same reason
	// DefaultOutputSchema copies: nothing downstream should be able to reach the
	// package default through a value we handed out.
	return DefaultOutputSchema(name)
}
