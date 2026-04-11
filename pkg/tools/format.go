package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/txn2/mcp-trino/pkg/client"
)

// validFormats lists the accepted output format values.
var validFormats = []string{"json", "csv", "markdown"}

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

// formatOutput renders a QueryResult in the requested format.
// The format must already be validated via validateFormat.
func formatOutput(result *client.QueryResult, format string) (string, error) {
	if format == "" {
		format = "json"
	}

	switch format {
	case "csv":
		return formatCSV(result), nil
	case "markdown":
		return formatMarkdown(result), nil
	default:
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(data), nil
	}
}

// tryUnwrapJSON attempts to unwrap a single-row, single-VARCHAR-column result
// containing a JSON string. Returns the parsed JSON and true if unwrapping
// succeeded, or nil and false if the result doesn't match the expected shape
// or the value isn't valid JSON.
func tryUnwrapJSON(result *client.QueryResult) (any, bool) {
	// Must be exactly one column
	if len(result.Columns) != 1 {
		return nil, false
	}

	// Column must be VARCHAR type
	if !strings.EqualFold(result.Columns[0].Type, "VARCHAR") &&
		!strings.HasPrefix(strings.ToLower(result.Columns[0].Type), "varchar(") {
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

	return parsed, true
}
