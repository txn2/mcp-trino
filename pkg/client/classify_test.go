package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/trinodb/trino-go-client/trino"
)

// failingTrino answers every statement with a FAILED result carrying errJSON,
// the "error" object of Trino's statement protocol, so the real driver builds
// the error value Classify sees.
func failingTrino(t *testing.T, errJSON string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := `{"id":"20261005_000000_00001_abcde","stats":{"state":"FAILED"},"error":` + errJSON + `}`
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})
}

// clientFor opens a Client through New against handler, so errors come from
// the real driver.
func clientFor(t *testing.T, handler http.Handler, timeout time.Duration) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return clientAt(t, srv.URL, timeout)
}

func clientAt(t *testing.T, rawURL string, timeout time.Duration) *Client {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{
		Host: u.Hostname(), Port: port, User: "test", Source: "test", Timeout: timeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// queryErr runs a query against c and returns the error the client reports.
func queryErr(t *testing.T, c *Client) error {
	t.Helper()
	_, err := c.Query(context.Background(), "SELECT 1", QueryOptions{})
	if err == nil {
		t.Fatal("query succeeded; the fixture should fail it")
	}
	return err
}

// trinoErrorJSON renders a Trino "error" object. Each cause, outermost first,
// is a Java exception type and message linked into the failureInfo cause chain.
func trinoErrorJSON(t *testing.T, errType, name string, code int, sqlState, message string, cause ...[2]string) string {
	t.Helper()
	var chain map[string]any
	for _, c := range slices.Backward(cause) {
		link := map[string]any{"type": c[0], "message": c[1]}
		if chain != nil {
			link["cause"] = chain
		}
		chain = link
	}
	failure := map[string]any{"type": "io.trino.spi.TrinoException", "message": message}
	if chain != nil {
		failure["cause"] = chain
	}
	obj := map[string]any{
		"message": message, "errorCode": code, "errorName": name, "errorType": errType,
		"failureInfo": failure,
	}
	if sqlState != "" {
		obj["sqlState"] = sqlState
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestClassify_TrinoErrors(t *testing.T) {
	const (
		jdbcError     = 0x0400_0000 // JdbcErrorCode.JDBC_ERROR
		pgConnFailed  = "The connection attempt failed."
		pgUniqueError = "ERROR: duplicate key value violates unique constraint \"t_pkey\""
	)

	tests := []struct {
		name      string
		errJSON   string
		category  ErrorCategory
		retryable bool
	}{
		{
			name: "USER_ERROR is client input",
			errJSON: trinoErrorJSON(t, "USER_ERROR", "TABLE_NOT_FOUND", 46, "",
				"line 40:10: Table 'scratch.uploads.t' does not exist"),
			category: CategoryClientInput,
		},
		{
			name: "INSUFFICIENT_RESOURCES is retryable upstream",
			errJSON: trinoErrorJSON(t, "INSUFFICIENT_RESOURCES", "EXCEEDED_GLOBAL_MEMORY_LIMIT", 131073, "",
				"Query exceeded distributed user memory limit"),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "INTERNAL_ERROR from a node shutting down is retryable upstream",
			errJSON: trinoErrorJSON(t, "INTERNAL_ERROR", "SERVER_SHUTTING_DOWN", 65545, "",
				"Server is shutting down"),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "INTERNAL_ERROR with no nodes available is retryable upstream",
			errJSON: trinoErrorJSON(t, "INTERNAL_ERROR", "NO_NODES_AVAILABLE", 65541, "",
				"No nodes available to run query"),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "INTERNAL_ERROR page transport is retryable upstream",
			errJSON: trinoErrorJSON(t, "INTERNAL_ERROR", "PAGE_TRANSPORT_TIMEOUT", 65540, "",
				"Encountered too many errors talking to a worker node"),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "other INTERNAL_ERROR is internal",
			errJSON: trinoErrorJSON(t, "INTERNAL_ERROR", "GENERIC_INTERNAL_ERROR", 65536, "",
				"Unexpected error"),
			category: CategoryInternal,
		},
		{
			name: "EXTERNAL with SQLSTATE class 08 is retryable upstream",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "08001",
				pgConnFailed),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "EXTERNAL with SQLSTATE class 23 is client input",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "23505",
				pgUniqueError),
			category: CategoryClientInput,
		},
		{
			name: "EXTERNAL with SQLSTATE class 22 is client input",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "22001",
				"value too long for type character varying(10)"),
			category: CategoryClientInput,
		},
		{
			name: "EXTERNAL with SQLSTATE class 42 is client input",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "42P01",
				"relation does not exist"),
			category: CategoryClientInput,
		},
		{
			// What a Trino server actually sends: sqlState null, the source's
			// failure only in the cause chain.
			name: "EXTERNAL caused by a refused connection is retryable upstream",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "", pgConnFailed,
				[2]string{"org.postgresql.util.PSQLException", pgConnFailed},
				[2]string{"java.net.ConnectException", "Connection refused"}),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "EXTERNAL with the PostgreSQL connect-failure message is retryable upstream",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "", pgConnFailed,
				[2]string{"org.postgresql.util.PSQLException", pgConnFailed}),
			category: CategoryUpstreamUnavailable, retryable: true,
		},
		{
			name: "EXTERNAL caused by a constraint violation is client input",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "", pgUniqueError,
				[2]string{"java.sql.SQLIntegrityConstraintViolationException", "Duplicate entry '1' for key 'PRIMARY'"}),
			category: CategoryClientInput,
		},
		{
			// A constraint violation outranks a connection type further down
			// the chain: retrying a rejected write is the costlier mistake.
			name: "EXTERNAL with both rejection and connection causes is client input",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "", "failed",
				[2]string{"java.sql.SQLSyntaxErrorException", "syntax error"},
				[2]string{"java.net.SocketException", "Connection reset"}),
			category: CategoryClientInput,
		},
		{
			name: "unrecognized EXTERNAL is upstream and not retryable",
			errJSON: trinoErrorJSON(t, "EXTERNAL", "JDBC_ERROR", jdbcError, "", pgUniqueError,
				[2]string{"org.postgresql.util.PSQLException", pgUniqueError}),
			category: CategoryUpstreamUnavailable,
		},
		{
			name: "unknown error type is internal",
			errJSON: trinoErrorJSON(t, "SOMETHING_NEW", "NEW_ERROR", 999, "",
				"from a future Trino"),
			category: CategoryInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var want trino.ErrTrino
			if err := json.Unmarshal([]byte(tt.errJSON), &want); err != nil {
				t.Fatal(err)
			}
			err := queryErr(t, clientFor(t, failingTrino(t, tt.errJSON), 10*time.Second))

			class, ok := Classify(err)
			if !ok {
				t.Fatalf("Classify(%v) reported no classification", err)
			}
			if class.Category != tt.category || class.Retryable != tt.retryable {
				t.Errorf("Classify = (%s, retryable %v), want (%s, retryable %v)",
					class.Category, class.Retryable, tt.category, tt.retryable)
			}
			if class.Transport != nil {
				t.Errorf("Transport = %+v, want nil for an error Trino reported", class.Transport)
			}
			wantDetail := TrinoErrorDetail{
				ErrorType: want.ErrorType, ErrorName: want.ErrorName, ErrorCode: want.ErrorCode,
				SQLState: want.SqlState, HTTPStatus: http.StatusOK,
			}
			if class.Trino == nil || *class.Trino != wantDetail {
				t.Errorf("Trino = %+v, want %+v", class.Trino, wantDetail)
			}
			if wantMsg := want.ErrorType + ": " + want.Message; class.Message != wantMsg {
				t.Errorf("Message = %q, want %q", class.Message, wantMsg)
			}
		})
	}
}

func TestClassify_UserCancelledIsNotClassified(t *testing.T) {
	errJSON := trinoErrorJSON(t, "USER_ERROR", "USER_CANCELLED", 3, "", "Query was canceled") //nolint:misspell // Trino's error name
	err := queryErr(t, clientFor(t, failingTrino(t, errJSON), 10*time.Second))
	if !errors.Is(err, trino.ErrQueryCancelled) {
		t.Fatalf("driver returned %v, want trino.ErrQueryCancelled", err)
	}
	if class, ok := Classify(err); ok {
		t.Errorf("Classify(%v) = %+v, want no classification", err, class)
	}
}

func TestClassify_TransportFromDriver(t *testing.T) {
	// The driver retries a failed dial with backoff (up to
	// trino.DefaultRequestRetryMaxAttempts or DefaultRequestRetryTimeout), so
	// a refused coordinator reaches the caller as the query deadline unless
	// those retries run out first; TestClassify_ErrorShapes covers that case.
	t.Run("connection refused until the deadline", func(t *testing.T) {
		var lc net.ListenConfig
		ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := "http://" + ln.Addr().String()
		if err := ln.Close(); err != nil {
			t.Fatal(err)
		}
		assertTransport(t, queryErr(t, clientAt(t, addr, 300*time.Millisecond)), TransportTimeout)
	})

	t.Run("deadline exceeded while waiting on the coordinator", func(t *testing.T) {
		release := make(chan struct{})
		hang := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-release:
			}
		})
		c := clientFor(t, hang, 200*time.Millisecond)
		t.Cleanup(func() { close(release) })
		assertTransport(t, queryErr(t, c), TransportTimeout)
	})

	t.Run("connection dropped mid-request", func(t *testing.T) {
		drop := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Error("response writer cannot hijack")
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
		})
		assertTransport(t, queryErr(t, clientFor(t, drop, 10*time.Second)), TransportConnectionReset)
	})
}

func assertTransport(t *testing.T, err error, want TransportKind) {
	t.Helper()
	class, ok := Classify(err)
	if !ok {
		t.Fatalf("Classify(%v) reported no classification", err)
	}
	if class.Category != CategoryUpstreamUnavailable || !class.Retryable {
		t.Errorf("Classify(%v) = (%s, retryable %v), want retryable upstream_unavailable",
			err, class.Category, class.Retryable)
	}
	if class.Trino != nil {
		t.Errorf("Trino = %+v, want nil for a transport failure", class.Trino)
	}
	if class.Transport == nil || class.Transport.Kind != want {
		t.Fatalf("Transport = %+v, want kind %q (error: %v)", class.Transport, want, err)
	}
	if class.Transport.Detail == "" {
		t.Error("Transport.Detail is empty")
	}
}

// TestClassify_ErrorShapes covers failures a loopback server cannot produce
// reliably. Each is built the way trino-go-client builds it: roundTrip wraps
// a failed http.Client.Do in ErrQueryFailed with no status, gives up on a
// retried 502/503/504 with ErrQueryFailed{StatusCode, errors.New(reason)},
// and reports any other non-200 response with its status and body.
func TestClassify_ErrorShapes(t *testing.T) {
	doErr := func(inner error) error {
		return fmt.Errorf("query failed: %w", &trino.ErrQueryFailed{
			Reason: &url.Error{Op: "Post", URL: "http://trino:8080/v1/statement", Err: inner},
		})
	}
	statusErr := func(status int, body string) error {
		return fmt.Errorf("query failed: %w", &trino.ErrQueryFailed{StatusCode: status, Reason: errors.New(body)})
	}

	tests := []struct {
		name       string
		err        error
		category   ErrorCategory
		retryable  bool
		kind       TransportKind // empty: no transport detail
		httpStatus int
	}{
		{
			name:     "DNS failure",
			err:      doErr(&net.DNSError{Err: "no such host", Name: "trino", IsNotFound: true}),
			category: CategoryUpstreamUnavailable, retryable: true, kind: TransportDNS,
		},
		{
			// The driver's give-up wraps the last dial error with %w.
			name: "connection refused after the driver's retries",
			err: fmt.Errorf("query failed: %w", &trino.ErrQueryFailed{Reason: fmt.Errorf(
				"giving up after 20 attempts in 1m50s (request_retry_timeout=2m0s, request_retry_max_attempts=20): %w",
				&url.Error{Op: "Post", URL: "http://trino:8080/v1/statement", Err: &net.OpError{
					Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED),
				}},
			)}),
			category: CategoryUpstreamUnavailable, retryable: true, kind: TransportConnectionRefused,
		},
		{
			name: "connection reset",
			err: doErr(&net.OpError{
				Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET),
			}),
			category: CategoryUpstreamUnavailable, retryable: true, kind: TransportConnectionReset,
		},
		{
			name:     "network timeout",
			err:      doErr(&net.OpError{Op: "dial", Net: "tcp", Err: timeoutErr{}}),
			category: CategoryUpstreamUnavailable, retryable: true, kind: TransportTimeout,
		},
		{
			name:     "other network operation error",
			err:      doErr(&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("no route to host")}),
			category: CategoryUpstreamUnavailable, retryable: true, kind: TransportNetwork,
		},
		{
			name: "gave up retrying 503",
			err: statusErr(http.StatusServiceUnavailable,
				"giving up after 5 attempts in 1m0s (request_retry_timeout=1m0s, request_retry_max_attempts=5)"),
			category: CategoryUpstreamUnavailable, retryable: true,
			kind: TransportHTTPStatus, httpStatus: http.StatusServiceUnavailable,
		},
		{
			name:     "gave up retrying 502",
			err:      statusErr(http.StatusBadGateway, "giving up"),
			category: CategoryUpstreamUnavailable, retryable: true,
			kind: TransportHTTPStatus, httpStatus: http.StatusBadGateway,
		},
		{
			name:     "gave up retrying 504",
			err:      statusErr(http.StatusGatewayTimeout, "giving up"),
			category: CategoryUpstreamUnavailable, retryable: true,
			kind: TransportHTTPStatus, httpStatus: http.StatusGatewayTimeout,
		},
		{
			name:     "429",
			err:      statusErr(http.StatusTooManyRequests, "slow down"),
			category: CategoryUpstreamUnavailable, retryable: true,
			kind: TransportHTTPStatus, httpStatus: http.StatusTooManyRequests,
		},
		{
			name:     "401 is not retryable",
			err:      statusErr(http.StatusUnauthorized, "Unauthorized"),
			category: CategoryInternal,
			kind:     TransportHTTPStatus, httpStatus: http.StatusUnauthorized,
		},
		{
			// newErrQueryFailedFromResponse keeps the status when reading the
			// body fails; the status must decide, not the read error.
			name:     "401 whose body read failed",
			err:      fmt.Errorf("query failed: %w", &trino.ErrQueryFailed{StatusCode: http.StatusUnauthorized, Reason: io.ErrUnexpectedEOF}),
			category: CategoryInternal,
			kind:     TransportHTTPStatus, httpStatus: http.StatusUnauthorized,
		},
		{
			name:     "error the client raised itself",
			err:      errors.New(`schema "s" needs a catalog: set QueryOptions.Catalog or Config.Catalog`),
			category: CategoryInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class, ok := Classify(tt.err)
			if !ok {
				t.Fatalf("Classify(%v) reported no classification", tt.err)
			}
			if class.Category != tt.category || class.Retryable != tt.retryable {
				t.Errorf("Classify = (%s, retryable %v), want (%s, retryable %v)",
					class.Category, class.Retryable, tt.category, tt.retryable)
			}
			if class.Trino != nil {
				t.Errorf("Trino = %+v, want nil", class.Trino)
			}
			if class.Message != tt.err.Error() {
				t.Errorf("Message = %q, want %q", class.Message, tt.err.Error())
			}
			if tt.kind == "" {
				if class.Transport != nil {
					t.Errorf("Transport = %+v, want nil", class.Transport)
				}
				return
			}
			if class.Transport == nil {
				t.Fatalf("Transport is nil, want kind %q", tt.kind)
			}
			if class.Transport.Kind != tt.kind || class.Transport.HTTPStatus != tt.httpStatus {
				t.Errorf("Transport = %+v, want kind %q status %d", class.Transport, tt.kind, tt.httpStatus)
			}
			var qf *trino.ErrQueryFailed
			if errors.As(tt.err, &qf) && class.Transport.Detail != qf.Reason.Error() {
				t.Errorf("Detail = %q, want the driver's reason %q", class.Transport.Detail, qf.Reason.Error())
			}
		})
	}
}

func TestClassify_NotClassified(t *testing.T) {
	for _, err := range []error{
		nil,
		context.Canceled,
		fmt.Errorf("query failed: %w", context.Canceled),
		trino.ErrQueryCancelled,
		fmt.Errorf("row iteration error: %w", trino.ErrQueryCancelled),
	} {
		if class, ok := Classify(err); ok {
			t.Errorf("Classify(%v) = %+v, want no classification", err, class)
		}
	}
}

// timeoutErr is a net.Error whose Timeout reports true, as a dial timeout's
// does.
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
