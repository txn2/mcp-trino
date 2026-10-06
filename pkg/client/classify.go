package client

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"

	"github.com/trinodb/trino-go-client/trino"
)

// ErrorCategory says whose problem a failed query is.
type ErrorCategory string

const (
	// CategoryUpstreamUnavailable is a failure of Trino, of a data source
	// behind it, or of the network between: nothing the caller sent was wrong.
	CategoryUpstreamUnavailable ErrorCategory = "upstream_unavailable"

	// CategoryClientInput is a failure caused by the statement itself: bad
	// SQL, a missing table, a constraint violation. Retrying it fails again.
	CategoryClientInput ErrorCategory = "client_input"

	// CategoryInternal is any other failure: a Trino bug, a misconfigured
	// client, an error this package does not recognize.
	CategoryInternal ErrorCategory = "internal"
)

// TransportKind names how a query failed when Trino never reported an error
// of its own.
type TransportKind string

const (
	// TransportTimeout is a deadline exceeded or a network timeout.
	TransportTimeout TransportKind = "timeout"
	// TransportConnectionRefused is a refused TCP connection.
	TransportConnectionRefused TransportKind = "connection_refused"
	// TransportConnectionReset is a connection closed or reset mid-request.
	TransportConnectionReset TransportKind = "connection_reset"
	// TransportDNS is a failed host name lookup.
	TransportDNS TransportKind = "dns"
	// TransportNetwork is any other network operation error.
	TransportNetwork TransportKind = "network"
	// TransportHTTPStatus is a non-200 HTTP response with no Trino error body.
	TransportHTTPStatus TransportKind = "http_status"
)

// ErrorClass is the classification of a failed query.
type ErrorClass struct {
	// Category says whose problem the failure is.
	Category ErrorCategory `json:"category"`

	// Retryable reports whether running the same statement again later is
	// expected to succeed.
	Retryable bool `json:"retryable"`

	// Message is Trino's own message when it reported one, otherwise the
	// error text.
	Message string `json:"message"`

	// Trino holds the error Trino reported. It is nil when Trino reported none.
	Trino *TrinoErrorDetail `json:"trino"`

	// Transport describes a failure that happened before Trino could report
	// an error. It is nil when Trino reported one, and for errors that are
	// neither.
	Transport *TransportErrorDetail `json:"transport"`
}

// TrinoErrorDetail is the error a Trino server reported for a query.
type TrinoErrorDetail struct {
	ErrorType  string `json:"error_type"`
	ErrorName  string `json:"error_name"`
	ErrorCode  int    `json:"error_code"`
	SQLState   string `json:"sql_state,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

// TransportErrorDetail describes a query that failed before Trino reported an
// error.
type TransportErrorDetail struct {
	Kind       TransportKind `json:"kind"`
	HTTPStatus int           `json:"http_status,omitempty"`
	Detail     string        `json:"detail"`
}

// Trino error types, from io.trino.spi.ErrorType.
const (
	trinoUserError             = "USER_ERROR"
	trinoInternalError         = "INTERNAL_ERROR"
	trinoInsufficientResources = "INSUFFICIENT_RESOURCES"
	trinoExternal              = "EXTERNAL"
)

// transientInternalErrors are the INTERNAL_ERROR names, from
// io.trino.spi.StandardErrorCode, that report a node or the transport between
// nodes failing rather than a defect: the same query is expected to pass once
// the cluster recovers.
var transientInternalErrors = map[string]bool{
	"TOO_MANY_REQUESTS_FAILED": true,
	"PAGE_TRANSPORT_ERROR":     true,
	"PAGE_TRANSPORT_TIMEOUT":   true,
	"NO_NODES_AVAILABLE":       true,
	"REMOTE_TASK_ERROR":        true,
	"SERVER_SHUTTING_DOWN":     true,
	"SERVER_STARTING_UP":       true,
	"ABANDONED_TASK":           true,
	"REMOTE_HOST_GONE":         true,
	"REMOTE_TASK_FAILED":       true,
}

// Java exception types in an EXTERNAL failure's cause chain. The Trino server
// always sends a null sqlState (it builds QueryError with null in
// io.trino.server.protocol.ProtocolUtil), so a connector's SQLSTATE reaches the
// client only as the JDBC exception subclass the driver chose for it. The
// java.sql types below are the ones the JDBC specification assigns to SQLSTATE
// classes 08, 22, 23 and 42.
var (
	unreachableSourceExceptions = map[string]bool{
		"java.net.ConnectException":                   true,
		"java.net.NoRouteToHostException":             true,
		"java.net.SocketException":                    true,
		"java.net.SocketTimeoutException":             true,
		"java.net.UnknownHostException":               true,
		"java.sql.SQLNonTransientConnectionException": true, // class 08
		"java.sql.SQLTransientConnectionException":    true, // class 08
		// Not tied to a SQLSTATE class: the specification defines it as a
		// failure that may succeed after the application reconnects.
		"java.sql.SQLRecoverableException": true,
	}

	clientInputExceptions = map[string]bool{
		"java.sql.SQLDataException":                         true, // class 22
		"java.sql.SQLIntegrityConstraintViolationException": true, // class 23
		"java.sql.SQLSyntaxErrorException":                  true, // class 42
	}

	// Messages a connector's driver uses when it cannot reach its source, for
	// drivers that raise a plain SQLException.
	// PostgreSQL's JDBC driver reports SQLSTATE 08001 with this text.
	unreachableSourceMessages = []string{
		"The connection attempt failed.",
	}
)

// Classify reports what kind of failure err is, and whether the same query is
// expected to succeed if run again. It understands the errors Client returns,
// which wrap trino-go-client's *trino.ErrQueryFailed.
//
// The second result is false when err is nil or a cancellation
// (trino.ErrQueryCancelled or context.Canceled): a canceled query did not
// fail, and the caller already knows why it stopped. Classify cannot tell a
// deadline on the caller's own context from a query timeout, since both
// surface as context.DeadlineExceeded and Classify reports a timeout. A caller
// that holds the context should check ctx.Err() before classifying.
func Classify(err error) (ErrorClass, bool) {
	if err == nil || errors.Is(err, trino.ErrQueryCancelled) || errors.Is(err, context.Canceled) {
		return ErrorClass{}, false
	}

	var queryFailed *trino.ErrQueryFailed
	hasQueryFailed := errors.As(err, &queryFailed)

	var trinoErr *trino.ErrTrino
	if errors.As(err, &trinoErr) {
		class := classifyTrino(trinoErr)
		class.Message = trinoErr.Error()
		class.Trino = &TrinoErrorDetail{
			ErrorType: trinoErr.ErrorType,
			ErrorName: trinoErr.ErrorName,
			ErrorCode: trinoErr.ErrorCode,
			SQLState:  trinoErr.SqlState,
		}
		if hasQueryFailed {
			class.Trino.HTTPStatus = queryFailed.StatusCode
		}
		return class, true
	}

	detail := err.Error()
	if hasQueryFailed && queryFailed.Reason != nil {
		detail = queryFailed.Reason.Error()
	}
	class := ErrorClass{Category: CategoryInternal, Message: err.Error()}

	// A response arrived, so its status decides, even when reading its body
	// then failed with an error transportKind would match.
	if hasQueryFailed && queryFailed.StatusCode != 0 {
		class.Category, class.Retryable = classifyHTTPStatus(queryFailed.StatusCode)
		class.Transport = &TransportErrorDetail{
			Kind:       TransportHTTPStatus,
			HTTPStatus: queryFailed.StatusCode,
			Detail:     detail,
		}
		return class, true
	}

	if kind, ok := transportKind(err); ok {
		class.Category = CategoryUpstreamUnavailable
		class.Retryable = true
		class.Transport = &TransportErrorDetail{Kind: kind, Detail: detail}
	}
	return class, true
}

// classifyTrino decides the category of an error Trino reported.
func classifyTrino(e *trino.ErrTrino) ErrorClass {
	switch e.ErrorType {
	case trinoUserError:
		return ErrorClass{Category: CategoryClientInput}
	case trinoInsufficientResources:
		return ErrorClass{Category: CategoryUpstreamUnavailable, Retryable: true}
	case trinoInternalError:
		if transientInternalErrors[e.ErrorName] {
			return ErrorClass{Category: CategoryUpstreamUnavailable, Retryable: true}
		}
		return ErrorClass{Category: CategoryInternal}
	case trinoExternal:
		return classifyExternal(e)
	default:
		return ErrorClass{Category: CategoryInternal}
	}
}

// classifyExternal decides an EXTERNAL error, which a connector raises for
// both an unreachable source and a statement the source rejected. The SQLSTATE
// class decides it when the server sent one; otherwise the exception types and
// messages in the failure's cause chain do. An EXTERNAL error neither
// recognizes is upstream but not retryable, so a permanent connector error is
// not retried forever.
func classifyExternal(e *trino.ErrTrino) ErrorClass {
	switch sqlStateClass(e.SqlState) {
	case "08":
		return ErrorClass{Category: CategoryUpstreamUnavailable, Retryable: true}
	case "22", "23", "42":
		return ErrorClass{Category: CategoryClientInput}
	default:
	}

	var rejected, unreachable bool
	for info := &e.FailureInfo; info != nil; info = info.Cause {
		rejected = rejected || clientInputExceptions[info.Type]
		unreachable = unreachable || unreachableSourceExceptions[info.Type] ||
			isUnreachableSourceMessage(info.Message)
	}
	switch {
	case rejected:
		return ErrorClass{Category: CategoryClientInput}
	case unreachable || isUnreachableSourceMessage(e.Message):
		return ErrorClass{Category: CategoryUpstreamUnavailable, Retryable: true}
	default:
		return ErrorClass{Category: CategoryUpstreamUnavailable}
	}
}

// sqlStateClass returns the two-character class of a SQLSTATE, or "" when
// state is too short to have one.
func sqlStateClass(state string) string {
	if len(state) < 2 {
		return ""
	}
	return state[:2]
}

func isUnreachableSourceMessage(msg string) bool {
	for _, m := range unreachableSourceMessages {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

// transportKind recognizes a network failure anywhere in err's chain.
func transportKind(err error) (TransportKind, bool) {
	var dnsErr *net.DNSError
	var netErr net.Error
	var opErr *net.OpError
	switch {
	case errors.As(err, &dnsErr):
		return TransportDNS, true
	case errors.Is(err, context.DeadlineExceeded),
		errors.As(err, &netErr) && netErr.Timeout():
		return TransportTimeout, true
	case errors.Is(err, syscall.ECONNREFUSED):
		return TransportConnectionRefused, true
	case errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE),
		errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return TransportConnectionReset, true
	case errors.As(err, &opErr):
		return TransportNetwork, true
	default:
		return "", false
	}
}

// classifyHTTPStatus decides a non-200 response that carried no Trino error.
// The driver retries 502, 503 and 504 itself and reports the status once it
// gives up; those and 429 say the coordinator, or a proxy in front of it, is
// not serving now. Any other status (a 401, a redirect the driver will not
// follow) is a client or deployment problem retrying will not fix.
func classifyHTTPStatus(status int) (ErrorCategory, bool) {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return CategoryUpstreamUnavailable, true
	default:
		return CategoryInternal, false
	}
}
