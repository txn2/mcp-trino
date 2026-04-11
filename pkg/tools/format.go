package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Output format identifiers.
const (
	outputFormatJSON     = "json"
	outputFormatCSV      = "csv"
	outputFormatMarkdown = "markdown"
)

// columnTypeJSON is the column type set when unwrapJSONColumn successfully
// parses a string column value into a JSON object or array.
const columnTypeJSON = "JSON"

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
// Must be kept in sync with the switch in handleExplain.
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

// formatOutput renders a QueryOutput in the requested format.
func formatOutput(qo *QueryOutput, format string) (string, error) {
	if format == "" {
		format = outputFormatJSON
	}

	switch format {
	case outputFormatCSV:
		return formatCSV(qo), nil
	case outputFormatMarkdown:
		return formatMarkdown(qo), nil
	case outputFormatJSON:
		data, err := json.MarshalIndent(qo, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported format %q", format)
	}
}

// formatCSV formats query output as CSV.
func formatCSV(qo *QueryOutput) string {
	if len(qo.Columns) == 0 {
		return ""
	}

	var output string

	// Header row
	for i, col := range qo.Columns {
		if i > 0 {
			output += ","
		}
		output += escapeCSV(col.Name)
	}
	output += "\n"

	// Data rows
	for _, row := range qo.Rows {
		for i, col := range qo.Columns {
			if i > 0 {
				output += ","
			}
			if val, ok := row[col.Name]; ok {
				output += escapeCSV(stringifyValue(val))
			}
		}
		output += "\n"
	}

	// Stats footer
	output += fmt.Sprintf("\n# %d rows returned", qo.Stats.RowCount)
	if qo.Stats.Truncated {
		output += fmt.Sprintf(" (truncated at limit %d)", qo.Stats.LimitApplied)
	}
	output += fmt.Sprintf(", executed in %dms", qo.Stats.DurationMs)

	return output
}

// formatMarkdown formats query output as a Markdown table.
func formatMarkdown(qo *QueryOutput) string {
	if len(qo.Columns) == 0 {
		return "No results"
	}

	var output string

	// Header row
	output += "|"
	for _, col := range qo.Columns {
		output += " " + col.Name + " |"
	}
	output += "\n|"

	// Separator row
	for range qo.Columns {
		output += " --- |"
	}
	output += "\n"

	// Data rows
	for _, row := range qo.Rows {
		output += "|"
		for _, col := range qo.Columns {
			val := ""
			if v, ok := row[col.Name]; ok && v != nil {
				val = stringifyValue(v)
			}
			output += " " + val + " |"
		}
		output += "\n"
	}

	// Stats footer
	output += fmt.Sprintf("\n*%d rows returned", qo.Stats.RowCount)
	if qo.Stats.Truncated {
		output += fmt.Sprintf(" (truncated at limit %d)", qo.Stats.LimitApplied)
	}
	output += fmt.Sprintf(", executed in %dms*", qo.Stats.DurationMs)

	return output
}

// stringifyValue converts a value to its string representation.
// For maps and slices (e.g. unwrapped JSON), it produces compact JSON.
// For all other types, it uses fmt.Sprintf.
func stringifyValue(v any) string {
	switch v.(type) {
	case map[string]any, []any:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
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

// unwrapJSONColumn attempts to unwrap a single-row, single-string-column
// QueryOutput whose value is a JSON object or array. On success it mutates
// the QueryOutput in place: the column type is changed to "JSON" and the
// row value is replaced with the parsed object. On failure the QueryOutput
// is left unchanged.
func unwrapJSONColumn(qo *QueryOutput) {
	// Must be exactly one column with a string-like type
	if len(qo.Columns) != 1 || !isStringColumnType(qo.Columns[0].Type) {
		return
	}

	// Must be exactly one row
	if len(qo.Rows) != 1 {
		return
	}

	colName := qo.Columns[0].Name
	val, ok := qo.Rows[0][colName]
	if !ok {
		return
	}

	str, ok := val.(string)
	if !ok {
		return
	}

	var parsed any
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return
	}

	// Only unwrap objects and arrays — scalar JSON values (strings, numbers,
	// booleans, null) are not meaningful to unwrap and would be confusing
	// or silently change types.
	switch parsed.(type) {
	case map[string]any, []any:
		qo.Columns[0].Type = columnTypeJSON
		qo.Rows[0][colName] = parsed
	default:
		// Fall through — leave QueryOutput unchanged.
	}
}
