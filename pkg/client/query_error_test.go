package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trinodb/trino-go-client/trino"
)

const runningQueryID = "20261009_000000_00001_runng"

// runningTrino accepts every statement as runningQueryID and never finishes
// it: it serves pages in order, then holds the next poll until the client
// gives up. It answers a cancel with cancelStatus.
type runningTrino struct {
	t            *testing.T
	pages        []string
	cancelStatus int
	cancels      atomic.Int32
}

func (f *runningTrino) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	next := func(n int) string {
		return `"nextUri":"http://` + r.Host + `/v1/statement/executing/` + runningQueryID + `/s/` + strconv.Itoa(n) + `"`
	}
	var body string
	switch r.Method {
	case http.MethodPost:
		body = `{"id":"` + runningQueryID + `","stats":{"state":"QUEUED"},` + next(1) + `}`
	case http.MethodGet:
		n, err := strconv.Atoi(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:])
		if err != nil || n > len(f.pages) {
			<-r.Context().Done()
			return
		}
		body = `{"id":"` + runningQueryID + `","stats":{"state":"RUNNING"},` + next(n+1) + `,` + f.pages[n-1] + `}`
	case http.MethodDelete:
		f.cancels.Add(1)
		w.WriteHeader(f.cancelStatus)
		return
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		f.t.Errorf("write response: %v", err)
	}
}

// onePage is a result page with one bigint row.
const onePage = `"columns":[{"name":"id","type":"bigint","typeSignature":{"rawType":"bigint","arguments":[]}}],"data":[[1]]`

func assertStatementTimeout(t *testing.T, err error, timeout time.Duration, wantConfirmed bool) {
	t.Helper()
	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("error %v (%T) is not a *QueryError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error %v does not unwrap to context.DeadlineExceeded", err)
	}
	if qe.QueryID != runningQueryID || !qe.Accepted {
		t.Errorf("QueryID %q Accepted %v, want %q and true", qe.QueryID, qe.Accepted, runningQueryID)
	}
	if !qe.CancelRequested || qe.CancelConfirmed != wantConfirmed {
		t.Errorf("CancelRequested %v CancelConfirmed %v, want true and %v",
			qe.CancelRequested, qe.CancelConfirmed, wantConfirmed)
	}
	if qe.Elapsed < timeout {
		t.Errorf("Elapsed %v, want at least the %v timeout", qe.Elapsed, timeout)
	}

	class, ok := Classify(err)
	if !ok {
		t.Fatalf("Classify(%v) reported no classification", err)
	}
	if class.Category != CategoryClientInput || class.Retryable {
		t.Errorf("Classify = (%s, retryable %v), want non-retryable client_input", class.Category, class.Retryable)
	}
	if class.Transport != nil || class.Trino != nil {
		t.Errorf("Transport %+v Trino %+v, want both nil for a statement timeout", class.Transport, class.Trino)
	}
	if class.QueryID != runningQueryID {
		t.Errorf("QueryID %q, want %q", class.QueryID, runningQueryID)
	}
	want := StatementTimeoutDetail{
		ElapsedMs: qe.Elapsed.Milliseconds(), CancelRequested: true, CancelConfirmed: wantConfirmed,
	}
	if class.StatementTimeout == nil || *class.StatementTimeout != want {
		t.Errorf("StatementTimeout %+v, want %+v", class.StatementTimeout, want)
	}
}

func TestQueryError_StatementTimeout(t *testing.T) {
	const timeout = 300 * time.Millisecond
	tests := []struct {
		name          string
		cancelStatus  int
		wantConfirmed bool
	}{
		{name: "cancel confirmed", cancelStatus: http.StatusNoContent, wantConfirmed: true},
		{name: "cancel refused", cancelStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator := &runningTrino{t: t, cancelStatus: tt.cancelStatus}
			c := clientFor(t, coordinator, timeout)

			cur, err := c.QueryStream(context.Background(), "CREATE TABLE t AS SELECT 1", QueryOptions{})
			if cur != nil {
				_ = cur.Close()
				t.Fatal("QueryStream returned a cursor for a statement that never produced a page")
			}
			// The text is what it was before the error was typed.
			if !strings.HasPrefix(err.Error(), "query failed: ") {
				t.Errorf("error text %q lost its prefix", err)
			}
			assertStatementTimeout(t, err, timeout, tt.wantConfirmed)
			if n := coordinator.cancels.Load(); n != 1 {
				t.Errorf("coordinator received %d cancels, want 1", n)
			}
		})
	}
}

func TestQueryError_StatementTimeoutMidStream(t *testing.T) {
	const timeout = 300 * time.Millisecond
	coordinator := &runningTrino{t: t, pages: []string{onePage}, cancelStatus: http.StatusNoContent}
	c := clientFor(t, coordinator, timeout)

	cur, err := c.QueryStream(context.Background(), "SELECT id", QueryOptions{})
	if err != nil {
		t.Fatalf("QueryStream: %v", err)
	}
	defer func() { _ = cur.Close() }()

	if !cur.Next() {
		t.Fatalf("Next found no first row: %v", cur.Err())
	}
	if cur.Next() {
		t.Fatal("Next returned a row the coordinator never sent")
	}
	assertStatementTimeout(t, cur.Err(), timeout, true)
	if n := coordinator.cancels.Load(); n != 1 {
		t.Errorf("coordinator received %d cancels, want 1", n)
	}
}

func TestQueryError_NotAccepted(t *testing.T) {
	release := make(chan struct{})
	hang := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	c := clientFor(t, hang, 200*time.Millisecond)
	t.Cleanup(func() { close(release) })

	err := queryErr(t, c)
	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("error %v (%T) is not a *QueryError", err, err)
	}
	if qe.Accepted || qe.QueryID != "" || qe.CancelRequested {
		t.Errorf("QueryError %+v, want no ID, not accepted, no cancel", qe)
	}
	class, _ := Classify(err)
	if class.StatementTimeout != nil || class.QueryID != "" {
		t.Errorf("Classify = %+v, want a transport timeout with no query ID", class)
	}
	assertTransport(t, err, TransportTimeout)
}

func TestQueryError_TrinoErrorCarriesQueryID(t *testing.T) {
	errJSON := trinoErrorJSON(t, "USER_ERROR", "TABLE_NOT_FOUND", 46, "", "Table 't' does not exist")
	err := queryErr(t, clientFor(t, failingTrino(t, errJSON), 10*time.Second))

	var qe *QueryError
	if !errors.As(err, &qe) {
		t.Fatalf("error %v (%T) is not a *QueryError", err, err)
	}
	const id = "20261005_000000_00001_abcde"
	if qe.QueryID != id || !qe.Accepted || qe.CancelRequested {
		t.Errorf("QueryError %+v, want ID %q, accepted, no cancel", qe, id)
	}
	class, _ := Classify(err)
	if class.QueryID != id || class.Trino == nil || class.StatementTimeout != nil {
		t.Errorf("Classify = %+v, want the Trino error with query ID %q", class, id)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTrackingTransport(t *testing.T) {
	errDial := errors.New("dial failed")
	tests := []struct {
		name          string
		method, path  string
		status        int
		err           error
		tracked       bool
		wantID        string
		wantRequested bool
		wantConfirmed bool
	}{
		{name: "statement accepted", method: http.MethodPost, path: "/v1/statement", status: http.StatusOK, tracked: true, wantID: "q1"},
		{name: "statement rejected", method: http.MethodPost, path: "/v1/statement", status: http.StatusServiceUnavailable, tracked: true},
		{name: "statement not sent", method: http.MethodPost, path: "/v1/statement", err: errDial, tracked: true},
		{name: "poll is not a submission", method: http.MethodGet, path: "/v1/statement/executing/q1/s/1",
			status: http.StatusOK, tracked: true},
		{name: "cancel confirmed", method: http.MethodDelete, path: "/v1/query/q1", status: http.StatusNoContent,
			tracked: true, wantRequested: true, wantConfirmed: true},
		{name: "cancel confirmed with 200", method: http.MethodDelete, path: "/v1/query/q1", status: http.StatusOK,
			tracked: true, wantRequested: true, wantConfirmed: true},
		{name: "cancel refused", method: http.MethodDelete, path: "/v1/query/q1", status: http.StatusNotFound,
			tracked: true, wantRequested: true},
		{name: "cancel not sent", method: http.MethodDelete, path: "/v1/query/q1", err: errDial, tracked: true, wantRequested: true},
		{name: "untracked request", method: http.MethodPost, path: "/v1/statement", status: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := roundTripFunc(func(*http.Request) (*http.Response, error) {
				if tt.err != nil {
					return nil, tt.err
				}
				return &http.Response{
					StatusCode: tt.status,
					Body:       io.NopCloser(strings.NewReader(`{"id":"q1","stats":{"state":"QUEUED"}}`)),
				}, nil
			})
			tracker := &queryTracker{}
			ctx := context.Background()
			if tt.tracked {
				ctx = withQueryTracker(ctx, tracker)
			}
			req, err := http.NewRequestWithContext(ctx, tt.method, "http://trino"+tt.path, http.NoBody)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := (&trackingTransport{base: base}).RoundTrip(req)
			if !errors.Is(err, tt.err) {
				t.Fatalf("RoundTrip error %v, want %v", err, tt.err)
			}
			if resp != nil {
				// The driver reads the body and closes it; the ID is taken then.
				if _, err := io.Copy(io.Discard, resp.Body); err != nil {
					t.Fatal(err)
				}
				if err := resp.Body.Close(); err != nil {
					t.Fatal(err)
				}
			}

			id, requested, confirmed := tracker.snapshot()
			if id != tt.wantID || requested != tt.wantRequested || confirmed != tt.wantConfirmed {
				t.Errorf("tracker = (%q, %v, %v), want (%q, %v, %v)",
					id, requested, confirmed, tt.wantID, tt.wantRequested, tt.wantConfirmed)
			}
		})
	}
}

func TestStatementBody_KeepsAtMostTheLimit(t *testing.T) {
	tests := []struct {
		name    string
		padding int
		wantID  string
	}{
		{name: "response within the limit", padding: 16, wantID: "q1"},
		{name: "response past the limit", padding: statementResponseLimit, wantID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"id":"q1","pad":"` + strings.Repeat("x", tt.padding) + `"}`
			tracker := &queryTracker{}
			b := &statementBody{ReadCloser: io.NopCloser(strings.NewReader(body)), tracker: tracker}

			read, err := io.ReadAll(b)
			if err != nil {
				t.Fatal(err)
			}
			if string(read) != body {
				t.Error("the driver did not read the body it was sent")
			}
			if b.buf.Len() > statementResponseLimit {
				t.Errorf("kept %d bytes, want at most %d", b.buf.Len(), statementResponseLimit)
			}
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
			if id, _, _ := tracker.snapshot(); id != tt.wantID {
				t.Errorf("query ID %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestNewQueryError_IDSources(t *testing.T) {
	errBoom := errors.New("boom")
	tracked := &queryTracker{queryID: "from-transport", cancelRequested: true, cancelConfirmed: true}
	progress := &queryProgressUpdater{}
	progress.Update(trino.QueryProgressInfo{QueryId: "from-callback"})

	tests := []struct {
		name          string
		tracker       *queryTracker
		progress      *queryProgressUpdater
		wantID        string
		wantConfirmed bool
	}{
		{name: "transport's ID wins", tracker: tracked, progress: progress, wantID: "from-transport", wantConfirmed: true},
		{name: "callback's ID when the transport saw none", tracker: &queryTracker{}, progress: progress, wantID: "from-callback"},
		{name: "callback's ID without a transport", progress: progress, wantID: "from-callback"},
		{name: "no ID at all", tracker: &queryTracker{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-time.Second)
			qe := newQueryError(errBoom, tt.tracker, tt.progress, start)
			if qe.QueryID != tt.wantID || qe.Accepted != (tt.wantID != "") || qe.CancelConfirmed != tt.wantConfirmed {
				t.Errorf("QueryError %+v, want ID %q, accepted %v, confirmed %v",
					qe, tt.wantID, tt.wantID != "", tt.wantConfirmed)
			}
			if qe.Elapsed < time.Second {
				t.Errorf("Elapsed %v, want it measured from start", qe.Elapsed)
			}
			if !errors.Is(qe, errBoom) || qe.Error() != errBoom.Error() {
				t.Errorf("QueryError %v does not carry %v", qe, errBoom)
			}
		})
	}
}
