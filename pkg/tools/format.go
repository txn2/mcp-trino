package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/txn2/mcp-trino/pkg/client"
)

// Output format identifiers.
const (
	outputFormatJSON     = "json"
	outputFormatCSV      = "csv"
	outputFormatMarkdown = "markdown"
)

// validFormats lists the accepted output format values.
var validFormats = []string{outputFormatJSON, outputFormatCSV, outputFormatMarkdown}

// validateFormat checks that the given format is valid.
// An empty string is always valid (defaults to "json").
// Returns an error naming the accepted values if the format is not recognized.
func validateFormat(format string) error {
	if format == "" {
		return nil
	}
	for _, v := range validFormats {
		if format == v {
			return nil
		}
	}
	return fmt.Errorf("invalid format %q: must be one of %s", format, strings.Join(validFormats, ", "))
}

// validExplainTypes lists the accepted explain type values.
var validExplainTypes = []string{"logical", "distributed", "io", "validate"}

// validateExplainType checks that the given explain type is valid.
// An empty string is always valid (defaults to "logical").
// Returns an error naming the accepted values if the type is not recognized.
func validateExplainType(explainType string) error {
	if explainType == "" {
		return nil
	}
	for _, v := range validExplainTypes {
		if explainType == v {
			return nil
		}
	}
	return fmt.Errorf("invalid explain type %q: must be one of %s",
		explainType, strings.Join(validExplainTypes, ", "))
}

// formatOutput renders a QueryResult in the requested format.
// The format must already be validated via validateFormat.
func formatOutput(result *client.QueryResult, format string) (string, error) {
	if format == "" {
		format = outputFormatJSON
	}

	switch format {
	case outputFormatCSV:
		return formatCSV(result), nil
	case outputFormatMarkdown:
		return formatMarkdown(result), nil
	case outputFormatJSON:
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
}

// renderTextContent produces the text output for a tool response.
// If unwrap succeeded and format is JSON, it renders the full QueryOutput
// (including unwrapped_result) so the LLM sees clean JSON in its context.
// Otherwise it renders the raw query result in the requested format.
func renderTextContent(result *client.QueryResult, queryOutput *QueryOutput, format string) (string, error) {
	if queryOutput.UnwrappedResult != nil && (format == "" || format == outputFormatJSON) {
		data, err := json.MarshalIndent(queryOutput, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(data), nil
	}
	return formatOutput(result, format)
}

// isStringColumnType returns true if the Trino type name is a string-like type
// that could contain JSON: VARCHAR, VARCHAR(N), CHAR, CHAR(N), or JSON.
func isStringColumnType(typeName string) bool {
	lower := strings.ToLower(typeName)
	if lower == "varchar" || lower == "char" || lower == "json" {
		return true
	}
	if strings.HasPrefix(lower, "varchar(") || strings.HasPrefix(lower, "char(") {
		return true
	}
	return false
}

// tryUnwrapJSON attempts to unwrap a single-row, single-string-column result
// containing a JSON object or array. Returns the parsed JSON and true if
// unwrapping succeeded, or nil and false if the result doesn't match the
// expected shape, the value isn't valid JSON, or the value is a JSON scalar.
func tryUnwrapJSON(result *client.QueryResult) (any, bool) {
	// Must be exactly one column
	if len(result.Columns) != 1 {
		return nil, false
	}

	// Column must be a string-like type (VARCHAR, CHAR, JSON)
	if !isStringColumnType(result.Columns[0].Type) {
		return nil, false
	}

	// Must be exactly one row
	if len(result.Rows) != 1 {
		return nil, false
	}

	// Get the column value
	val, ok := result.Rows[0][result.Columns[0].Name]
	if !ok {
		return nil, false
	}

	str, ok := val.(string)
	if !ok {
		return nil, false
	}

	// Try to parse as JSON
	var parsed any
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return nil, false
	}

	// Only unwrap objects and arrays — scalar JSON values (strings, numbers,
	// booleans, null) are not meaningful to unwrap and would be confusing
	// or silently change types.
	switch parsed.(type) {
	case map[string]any, []any:
		return parsed, true
	default:
		return nil, false
	}
}
