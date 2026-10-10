package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// QueryError is a query that failed after it was sent, with what the
// coordinator reported about it. Errors.Is and errors.As reach the driver's
// error through it.
type QueryError struct {
	// QueryID is the ID the coordinator assigned to the statement, or empty
	// when it assigned none before the failure.
	QueryID string

	// Accepted reports whether the coordinator accepted the statement, by
	// answering its submission with a query ID, before the failure. When
	// false, the statement most likely never reached the coordinator, but a
	// submission whose answer was lost may have created a query.
	Accepted bool

	// Elapsed is the time from the start of the call until the failure was
	// reported, including the driver's cancel request.
	Elapsed time.Duration

	// CancelRequested reports whether a cancel was sent to the coordinator
	// for the query. The driver sends one when the query's context ends
	// after the statement was accepted.
	CancelRequested bool

	// CancelConfirmed reports whether the coordinator answered that cancel
	// with success. A query whose cancel was requested but not confirmed
	// may still be running.
	CancelConfirmed bool

	// Err is the failure.
	Err error
}

// Error returns the failure's text.
func (e *QueryError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the failure.
func (e *QueryError) Unwrap() error {
	return e.Err
}

// statementResponseLimit caps how much of a statement submission's response
// is kept to read the query ID from. The coordinator answers a submission with
// the query's first, data-free result, far smaller than this.
const statementResponseLimit = 1 << 20

// queryTracker records what the coordinator answered for one query: the ID it
// assigned when it accepted the statement, and how it answered a cancel.
type queryTracker struct {
	mu              sync.Mutex
	queryID         string
	cancelRequested bool
	cancelConfirmed bool
}

type queryTrackerKey struct{}

// withQueryTracker returns ctx carrying t, which trackingTransport fills in
// from the requests the driver sends with ctx.
func withQueryTracker(ctx context.Context, t *queryTracker) context.Context {
	return context.WithValue(ctx, queryTrackerKey{}, t)
}

func queryTrackerFrom(ctx context.Context) *queryTracker {
	if t, ok := ctx.Value(queryTrackerKey{}).(*queryTracker); ok {
		return t
	}
	return nil
}

// recordStatement reads the query ID from a statement submission's response
// body. A body cut short, by the limit or a failed read, carries no ID.
func (t *queryTracker) recordStatement(body []byte) {
	var resp struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &resp) != nil || resp.ID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.queryID = resp.ID
}

// recordCancel notes a cancel the driver sent and whether the coordinator
// confirmed it. Trino answers a cancel with 204 No Content.
func (t *queryTracker) recordCancel(resp *http.Response, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelRequested = true
	t.cancelConfirmed = err == nil &&
		(resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK)
}

// snapshot returns the query ID and cancel state recorded so far.
func (t *queryTracker) snapshot() (queryID string, cancelRequested, cancelConfirmed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.queryID, t.cancelRequested, t.cancelConfirmed
}

// trackingTransport records, in the queryTracker a request's context carries,
// the query ID from a statement submission's response and the outcome of a
// cancel. The driver's progress callback cannot stand in for it: the driver
// drops the callback's first update when its receiver is not yet waiting, and
// sends no other until a result page arrives, so a query whose deadline fires
// before then would have no ID. Requests without a tracker pass through.
type trackingTransport struct {
	base http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *trackingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tracker := queryTrackerFrom(req.Context())
	resp, err := t.base.RoundTrip(req)
	if tracker == nil {
		return resp, err
	}
	switch {
	case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/v1/statement"):
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body = &statementBody{ReadCloser: resp.Body, tracker: tracker}
		}
	case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/v1/query/"):
		tracker.recordCancel(resp, err)
	default:
	}
	return resp, err
}

// statementBody keeps what the driver reads of a statement submission's
// response, and hands it to the tracker when the driver closes it.
type statementBody struct {
	io.ReadCloser
	tracker *queryTracker
	buf     bytes.Buffer
}

func (b *statementBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if room := statementResponseLimit - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(n, room)])
	}
	return n, err
}

func (b *statementBody) Close() error {
	b.tracker.recordStatement(b.buf.Bytes())
	return b.ReadCloser.Close()
}

// newQueryError returns err as a *QueryError, with the query ID and cancel
// state the tracker recorded. Without a tracker's ID (a client made with
// NewWithDB has no tracking transport), it falls back to the ID the progress
// callback captured, and a statement with an ID counts as accepted.
func newQueryError(err error, tracker *queryTracker, progress *queryProgressUpdater, start time.Time) *QueryError {
	qe := &QueryError{Elapsed: time.Since(start), Err: err}
	if tracker != nil {
		qe.QueryID, qe.CancelRequested, qe.CancelConfirmed = tracker.snapshot()
	}
	if qe.QueryID == "" && progress != nil {
		qe.QueryID = progress.QueryID()
	}
	qe.Accepted = qe.QueryID != ""
	return qe
}
