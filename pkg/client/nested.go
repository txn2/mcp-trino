package client

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/trinodb/trino-go-client/trino"
)

// typeKind is what rendering a value needs to know of its Trino type.
type typeKind int

const (
	kindOther typeKind = iota
	kindArray
	kindMap
	kindRow
	kindDate
	kindTime
	kindTimestamp
)

// defaultTimePrecision is the precision Trino gives a TIME or TIMESTAMP that
// declares none.
const defaultTimePrecision = 3

// valueType is a column type parsed from the name the driver reports for it,
// such as "ARRAY(TIMESTAMP(3))" or "ROW(A BIGINT, B DATE)".
type valueType struct {
	kind      typeKind
	precision int  // fractional-second digits of a TIME or TIMESTAMP
	zoned     bool // TIME or TIMESTAMP WITH TIME ZONE
	// args are an ARRAY's element type, a MAP's key and value types, or a
	// ROW's field types in order.
	args []*valueType
}

// parseValueType parses a type name as the driver reports it: upper case,
// with the full type of an ARRAY, MAP or ROW. A name it does not recognize
// parses as kindOther.
func parseValueType(name string) *valueType {
	name = strings.TrimSpace(name)
	switch {
	case strings.HasPrefix(name, "ARRAY("):
		return containerType(kindArray, name, identity)
	case strings.HasPrefix(name, "MAP("):
		return containerType(kindMap, name, identity)
	case strings.HasPrefix(name, "ROW("):
		return containerType(kindRow, name, rowFieldType)
	case name == "DATE":
		return &valueType{kind: kindDate}
	}
	return temporalType(name)
}

// containerType parses the argument types of an ARRAY, MAP or ROW, where
// argType takes the type out of each argument.
func containerType(kind typeKind, name string, argType func(string) string) *valueType {
	args := splitTypeArgs(typeArgs(name))
	t := &valueType{kind: kind, args: make([]*valueType, len(args))}
	for i, arg := range args {
		t.args[i] = parseValueType(argType(arg))
	}
	return t
}

func identity(s string) string { return s }

// temporalType parses TIME and TIMESTAMP, with an optional precision and
// time zone, and returns kindOther for any other name.
func temporalType(name string) *valueType {
	base, zoned := strings.CutSuffix(name, " WITH TIME ZONE")
	t := &valueType{precision: defaultTimePrecision, zoned: zoned}
	switch {
	case strings.HasPrefix(base, "TIMESTAMP"):
		t.kind, base = kindTimestamp, base[len("TIMESTAMP"):]
	case strings.HasPrefix(base, "TIME"):
		t.kind, base = kindTime, base[len("TIME"):]
	default:
		return &valueType{}
	}
	if base == "" {
		return t
	}
	digits, opened := strings.CutPrefix(base, "(")
	digits, closed := strings.CutSuffix(digits, ")")
	precision, err := strconv.Atoi(digits)
	if !opened || !closed || err != nil {
		return &valueType{}
	}
	t.precision = precision
	return t
}

// typeArgs returns what is between the outermost parentheses of name.
func typeArgs(name string) string {
	start, end := strings.IndexByte(name, '('), strings.LastIndexByte(name, ')')
	if start < 0 || end < start {
		return ""
	}
	return name[start+1 : end]
}

// splitTypeArgs splits a type's arguments at the commas outside parentheses
// and quoted field names.
func splitTypeArgs(args string) []string {
	var parts []string
	depth, quoted, start := 0, false, 0
	for i := 0; i < len(args); i++ {
		switch c := args[i]; {
		case c == '"':
			// A doubled quote inside a quoted name toggles twice.
			quoted = !quoted
		case quoted:
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == ',' && depth == 0:
			parts = append(parts, strings.TrimSpace(args[start:i]))
			start = i + 1
		}
	}
	if rest := strings.TrimSpace(args[start:]); rest != "" || len(parts) > 0 {
		parts = append(parts, rest)
	}
	return parts
}

// rowFieldType returns the type of a ROW field, which is a type alone for an
// anonymous field and a name followed by a type otherwise. Trino quotes a
// name that is not a plain identifier, so an unquoted name holds no space or
// parenthesis, and the only types with a space before any parenthesis are
// the INTERVAL types and an unparameterized TIME or TIMESTAMP WITH TIME ZONE.
func rowFieldType(field string) string {
	if strings.HasPrefix(field, `"`) {
		for i := 1; i < len(field); i++ {
			if field[i] != '"' {
				continue
			}
			if i+1 < len(field) && field[i+1] == '"' {
				i++
				continue
			}
			return strings.TrimSpace(field[i+1:])
		}
		return ""
	}
	space := strings.IndexByte(field, ' ')
	if space < 0 {
		return field
	}
	if paren := strings.IndexByte(field, '('); paren >= 0 && paren < space {
		return field
	}
	switch field {
	case "INTERVAL DAY TO SECOND", "INTERVAL YEAR TO MONTH", "TIME WITH TIME ZONE", "TIMESTAMP WITH TIME ZONE":
		return field
	}
	return field[space+1:]
}

// arg returns the i-th argument type, or kindOther when the type name did
// not say.
func (t *valueType) arg(i int) *valueType {
	if i < len(t.args) && t.args[i] != nil {
		return t.args[i]
	}
	return &valueType{}
}

// render converts a column value to the JSON-friendly form a result row
// carries. An ARRAY, MAP or ROW carries its elements as Trino writes them in
// its JSON response, so a nested DATE stays "2026-10-16" rather than gaining
// the client's zone; every other value goes through convertValue.
// The driver's value, not the parsed type, decides, so a container whose type
// name did not parse still renders, with kindOther elements.
func (t *valueType) render(v any) any {
	switch v.(type) {
	case []any, map[string]any, trino.Row:
		return t.nested(v)
	default:
		return convertValue(v)
	}
}

// nested converts a value inside an ARRAY, MAP or ROW, or the container
// itself. A ROW becomes the array of its field values in order.
func (t *valueType) nested(v any) any {
	switch val := v.(type) {
	case []any:
		out := make([]any, len(val))
		for i, element := range val {
			out[i] = t.arg(0).nested(element)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(val))
		for key, value := range val {
			out[key] = t.arg(1).nested(value)
		}
		return out
	case trino.Row:
		if !val.Valid {
			return nil
		}
		out := make([]any, val.Len())
		for i := range out {
			out[i] = t.arg(i).nested(val.Value(i))
		}
		return out
	case float64:
		return nonFinite(val)
	case time.Time:
		return t.formatTime(val)
	case []byte:
		return base64.StdEncoding.EncodeToString(val)
	default:
		return v
	}
}

// nonFinite returns the string Trino sends for a NaN or infinite DOUBLE or
// REAL, which JSON has no number for, and any other value unchanged.
func nonFinite(f float64) any {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	default:
		return f
	}
}

// formatTime writes tm the way Trino writes a value of type t. Trino allows
// twelve fractional digits and Go holds nine, so a TIME(12) or TIMESTAMP(12)
// shows the nine the driver kept.
//
// TODO: the driver reads a zoneless value in the session zone, so a
// TIMESTAMP whose wall clock falls in that zone's daylight-saving gap is
// moved out of it, to an instant a valid wall clock also names; which of the
// two Trino sent cannot be told apart here. A top-level TIMESTAMP is moved
// the same way.
func (t *valueType) formatTime(tm time.Time) string {
	switch t.kind {
	case kindDate:
		return dateOf(tm)
	case kindTime:
		s := tm.Format(time.TimeOnly) + fraction(tm, t.precision)
		if t.zoned {
			s += tm.Format("-07:00")
		}
		return s
	case kindTimestamp:
		s := tm.Format(time.DateTime) + fraction(tm, t.precision)
		if t.zoned {
			s += " " + zoneName(tm)
		}
		return s
	default:
		return tm.Format(time.RFC3339Nano)
	}
}

// dateOf returns the date the driver read as midnight in its session zone.
// Where that zone skips midnight, as America/Santiago does when daylight
// saving starts, the driver lands within a few hours of it, so the nearest
// midnight on tm's wall clock is the date Trino sent.
func dateOf(tm time.Time) string {
	wall := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(), time.UTC)
	return wall.Add(12 * time.Hour).Format(time.DateOnly)
}

// fraction returns the fractional seconds of tm to precision digits, with
// the leading dot, or "" for a precision of zero.
func fraction(tm time.Time, precision int) string {
	digits := min(precision, 9)
	if digits <= 0 {
		return ""
	}
	return "." + fmt.Sprintf("%09d", tm.Nanosecond())[:digits]
}

// zoneName returns the zone Trino named for tm: the zone ID the driver
// loaded, or the offset when Trino sent one. The driver parses an offset
// into an unnamed zone, or into time.Local when the offset is Local's.
func zoneName(tm time.Time) string {
	if loc := tm.Location(); loc != time.Local && loc.String() != "" {
		return loc.String()
	}
	return tm.Format("-07:00")
}
