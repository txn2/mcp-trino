package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trinodb/trino-go-client/trino"
)

// testdata/nested_trino_453.json holds the columns and data of a response
// recorded from trinodb/trino:453 for
//
//	SELECT CAST(ROW(1, 'two', 3.5e0) AS ROW(a BIGINT, b VARCHAR, c DOUBLE)) AS r,
//	  ARRAY[DATE '2026-10-16', NULL] AS dates,
//	  ARRAY[TIMESTAMP '2026-10-16 12:34:56.100'] AS ts,
//	  MAP(ARRAY['k'], ARRAY[TIMESTAMP '2026-10-16 12:34:56.123']) AS m,
//	  CAST(ROW(DATE '2026-10-16', ARRAY[X'0001FF'])
//	    AS ROW("Odd Name" DATE, b ARRAY(VARBINARY))) AS odd,
//	  ARRAY[TIMESTAMP '2026-10-16 12:34:56.123 America/New_York',
//	    TIMESTAMP '2026-10-16 12:34:56.123 +05:30'] AS tz,
//	  ARRAY[TIME '12:34:56.123', TIME '12:34:56.123 +05:30'] AS t,
//	  CAST(NULL AS ROW(a BIGINT)) AS nr

// nestedTrino serves the recorded response, from a second page as Trino does.
func nestedTrino(t *testing.T) http.Handler {
	t.Helper()
	recorded, err := os.ReadFile(filepath.Join("testdata", "nested_trino_453.json"))
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]any
	if err = json.Unmarshal(recorded, &page); err != nil {
		t.Fatal(err)
	}
	page["id"] = "q1"
	page["stats"] = map[string]any{"state": "FINISHED"}
	finished, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := finished
		if r.Method == http.MethodPost {
			body = []byte(`{"id":"q1","stats":{"state":"QUEUED"},"nextUri":"http://` + r.Host + `/v1/statement/q1/1"}`)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(body); err != nil {
			t.Errorf("write response: %v", err)
		}
	})
}

// TestQuery_NestedValues covers #113: through the real driver, a ROW carries
// its field values and a temporal value inside an ARRAY, MAP or ROW carries
// the text Trino sent, whatever the client's local zone.
func TestQuery_NestedValues(t *testing.T) {
	c := clientFor(t, nestedTrino(t), 10*time.Second)
	result, err := c.Query(context.Background(), "SELECT 1", DefaultQueryOptions())
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(result.Rows))
	}

	want := map[string]string{
		"r":     `[1,"two",3.5]`,
		"dates": `["2026-10-16",null]`,
		"ts":    `["2026-10-16 12:34:56.100"]`,
		"m":     `{"k":"2026-10-16 12:34:56.123"}`,
		"odd":   `["2026-10-16",["AAH/"]]`,
		"tz":    `["2026-10-16 12:34:56.123 America/New_York","2026-10-16 12:34:56.123 +05:30"]`,
		"t":     `["12:34:56.123+00:00","12:34:56.123+05:30"]`,
		"nr":    `null`,
	}
	for name, wantJSON := range want {
		got, err := json.Marshal(result.Rows[0][name])
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if string(got) != wantJSON {
			t.Errorf("%s: got %s, want %s", name, got, wantJSON)
		}
	}
}

// TestQuery_NestedRawValues pins that RawValues still returns the driver's
// values for a container, as its documentation says.
func TestQuery_NestedRawValues(t *testing.T) {
	c := clientFor(t, nestedTrino(t), 10*time.Second)
	result, err := c.Query(context.Background(), "SELECT 1", QueryOptions{Limit: 1, RawValues: true})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	row, ok := result.Rows[0]["r"].(trino.Row)
	if !ok || row.Len() != 3 || row.Value(1) != "two" {
		t.Errorf("raw ROW: got %#v, want a trino.Row of three fields", result.Rows[0]["r"])
	}
	dates, ok := result.Rows[0]["dates"].([]any)
	if !ok || len(dates) != 2 {
		t.Fatalf("raw ARRAY(DATE): got %#v", result.Rows[0]["dates"])
	}
	if _, ok := dates[0].(time.Time); !ok {
		t.Errorf("raw DATE element: got %T, want time.Time", dates[0])
	}
}

// describeType writes t compactly, so a parse result reads as one string.
func describeType(t *valueType) string {
	if t == nil {
		return "nil"
	}
	var b strings.Builder
	switch t.kind {
	case kindArray:
		b.WriteString("array")
	case kindMap:
		b.WriteString("map")
	case kindRow:
		b.WriteString("row")
	case kindDate:
		return "date"
	case kindTime:
		b.WriteString("time")
	case kindTimestamp:
		b.WriteString("timestamp")
	default:
		return "other"
	}
	if t.kind == kindTime || t.kind == kindTimestamp {
		fmt.Fprintf(&b, "(%d)", t.precision)
		if t.zoned {
			b.WriteString("tz")
		}
		return b.String()
	}
	args := make([]string, len(t.args))
	for i, arg := range t.args {
		args[i] = describeType(arg)
	}
	b.WriteString("[" + strings.Join(args, ",") + "]")
	return b.String()
}

func TestParseValueType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"BIGINT", "other"},
		{"DATE", "date"},
		{"TIMESTAMP", "timestamp(3)"},
		{"TIMESTAMP(0)", "timestamp(0)"},
		{"TIMESTAMP(12) WITH TIME ZONE", "timestamp(12)tz"},
		{"TIMESTAMP WITH TIME ZONE", "timestamp(3)tz"},
		{"TIME(6)", "time(6)"},
		{"TIME WITH TIME ZONE", "time(3)tz"},
		{"TIMESTAMP(X)", "other"},
		{"TIMESTAMP(3", "other"},
		{"TIME3)", "other"},
		{"ARRAY(DATE)", "array[date]"},
		{"ARRAY(ARRAY(TIMESTAMP(6)))", "array[array[timestamp(6)]]"},
		{"MAP(VARCHAR(1), TIMESTAMP(3))", "map[other,timestamp(3)]"},
		{"MAP(DECIMAL(3, 2), DATE)", "map[other,date]"},
		{"ROW(A BIGINT, B VARCHAR, C DOUBLE)", "row[other,other,other]"},
		{"ROW(D DATE, TS TIMESTAMP(3))", "row[date,timestamp(3)]"},
		{"ROW(DATE DATE, TIME TIME(3))", "row[date,time(3)]"},
		{"ROW(INTEGER, DATE)", "row[other,date]"},
		{"ROW(DECIMAL(3, 2), TIMESTAMP(3) WITH TIME ZONE)", "row[other,timestamp(3)tz]"},
		{"ROW(TIMESTAMP WITH TIME ZONE, INTERVAL DAY TO SECOND)", "row[timestamp(3)tz,other]"},
		{"ROW(T TIMESTAMP WITH TIME ZONE)", "row[timestamp(3)tz]"},
		{`ROW("ODD NAME" ARRAY(DATE), M MAP(VARCHAR, DATE))`, "row[array[date],map[other,date]]"},
		{`ROW("A, ""B"" (C)" DATE, D DATE)`, "row[date,date]"},
		{`ROW("UNTERMINATED DATE)`, "row[other]"},
		{"ARRAY(ROW(D DATE))", "array[row[date]]"},
		{"ARRAY()", "array[]"},
		{"ARRAY(DATE", "array[]"},
		{"ARRAY", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeType(parseValueType(tt.name)); got != tt.want {
				t.Errorf("parseValueType(%q) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// TestRenderNested covers values the recorded response does not hold.
func TestRenderNested(t *testing.T) {
	ts := time.Date(2026, 10, 16, 12, 34, 56, 123456789, time.UTC)
	tests := []struct {
		name  string
		typ   string
		value any
		want  string
	}{
		{"timestamp precision 0", "ARRAY(TIMESTAMP(0))", []any{ts}, `["2026-10-16 12:34:56"]`},
		{"timestamp precision 6", "ARRAY(TIMESTAMP(6))", []any{ts}, `["2026-10-16 12:34:56.123456"]`},
		{"timestamp precision 12 keeps nine digits", "ARRAY(TIMESTAMP(12))", []any{ts}, `["2026-10-16 12:34:56.123456789"]`},
		{"named zone", "ARRAY(TIMESTAMP(3) WITH TIME ZONE)", []any{ts}, `["2026-10-16 12:34:56.123 UTC"]`},
		{"unnamed offset", "ARRAY(TIMESTAMP(3) WITH TIME ZONE)", []any{ts.In(time.FixedZone("", -7*3600))}, `["2026-10-16 05:34:56.123 -07:00"]`},
		{"time without zone", "ARRAY(TIME(3))", []any{ts}, `["12:34:56.123"]`},
		{"time in an unknown type", "ARRAY(SOMETHING)", []any{ts}, `["2026-10-16T12:34:56.123456789Z"]`},
		{"varbinary element", "ARRAY(VARBINARY)", []any{[]byte{0, 1, 0xff}}, `["AAH/"]`},
		{"invalid row", "ARRAY(ROW(A BIGINT))", []any{trino.Row{}}, `[null]`},
		{"unparsed element type", "ARRAY(ROW)", []any{[]any{ts}}, `[["2026-10-16T12:34:56.123456789Z"]]`},
		{"NaN element", "ARRAY(DOUBLE)", []any{math.NaN()}, `["NaN"]`},
		{"infinite elements", "ARRAY(REAL)", []any{math.Inf(1), math.Inf(-1)}, `["Infinity","-Infinity"]`},
		{"finite element", "ARRAY(DOUBLE)", []any{1.5}, `[1.5]`},
		{"container under an unparsed type name", "", []any{ts}, `["2026-10-16T12:34:56.123456789Z"]`},
		{"top-level scalar unchanged", "TIMESTAMP", ts, `"2026-10-16T12:34:56.123456789Z"`},
		{"top-level varbinary unchanged", "VARBINARY", []byte("text"), `"text"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(resultValue(tt.value, parseValueType(tt.typ), false))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

// TestDateOf covers a DATE whose midnight the session zone skips: the driver
// reads America/Santiago's 2026-09-06 as 23:00 the evening before.
func TestDateOf(t *testing.T) {
	santiago, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Skipf("no zone database: %v", err)
	}
	read, err := time.ParseInLocation(time.DateOnly, "2026-09-06", santiago)
	if err != nil {
		t.Fatal(err)
	}
	if read.Day() == 6 && read.Hour() == 0 {
		t.Fatalf("zone database has no midnight gap on 2026-09-06: %v", read)
	}
	tests := []struct {
		in   time.Time
		want string
	}{
		{read, "2026-09-06"},
		{time.Date(2026, 9, 6, 1, 0, 0, 0, santiago), "2026-09-06"},
		{time.Date(2026, 10, 16, 0, 0, 0, 0, santiago), "2026-10-16"},
	}
	for _, tt := range tests {
		if got := dateOf(tt.in); got != tt.want {
			t.Errorf("dateOf(%v) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestZoneName(t *testing.T) {
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, time.Local)
	if got, want := zoneName(at), at.Format("-07:00"); got != want {
		t.Errorf("Local: got %q, want the offset %q", got, want)
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no zone database: %v", err)
	}
	if got := zoneName(at.In(ny)); got != "America/New_York" {
		t.Errorf("named zone: got %q", got)
	}
}
