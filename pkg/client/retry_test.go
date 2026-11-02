package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trinodb/trino-go-client/trino"
)

// unavailableTrino answers every request with 503, which trino-go-client
// retries, and counts the statements submitted to it.
type unavailableTrino struct {
	statements atomic.Int32
}

func (u *unavailableTrino) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/v1/statement" {
		u.statements.Add(1)
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

// newUnavailableClient opens a client through New against a server that is
// always unavailable, with the given retry limits.
func newUnavailableClient(t *testing.T, retryTimeout time.Duration, maxAttempts int) (*Client, *unavailableTrino) {
	t.Helper()
	fake := &unavailableTrino{}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{
		Host: u.Hostname(), Port: port, User: "test", Source: "test",
		Timeout:                 30 * time.Second,
		RequestRetryTimeout:     retryTimeout,
		RequestRetryMaxAttempts: maxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, fake
}

func TestQuery_RequestRetryMaxAttempts(t *testing.T) {
	tests := []struct {
		name        string
		maxAttempts int
	}{
		{name: "retries disabled", maxAttempts: 1},
		{name: "three attempts", maxAttempts: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, fake := newUnavailableClient(t, 30*time.Second, tt.maxAttempts)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, err := c.Query(ctx, "SELECT 1", QueryOptions{})
			if err == nil {
				t.Fatal("Query succeeded against a server that only answers 503")
			}
			if ctx.Err() != nil {
				t.Fatalf("Query ran until the context deadline (%v); the attempt limit was not applied", err)
			}
			if got := fake.statements.Load(); got != int32(tt.maxAttempts) {
				t.Errorf("statement sent %d times, want %d", got, tt.maxAttempts)
			}
			want := "request_retry_max_attempts=" + strconv.Itoa(tt.maxAttempts)
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not report %s", err, want)
			}
		})
	}
}

func TestQuery_RequestRetryTimeout(t *testing.T) {
	// Max attempts left at zero takes the driver's default of 20, so the
	// timeout is the limit that ends the retries.
	c, fake := newUnavailableClient(t, 500*time.Millisecond, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	start := time.Now()
	_, err := c.Query(ctx, "SELECT 1", QueryOptions{})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Query succeeded against a server that only answers 503")
	}
	if elapsed > 10*time.Second {
		t.Errorf("Query took %s; the 500ms retry timeout was not applied", elapsed)
	}
	if got := fake.statements.Load(); got < 2 || got >= int32(trino.DefaultRequestRetryMaxAttempts) {
		t.Errorf("statement sent %d times, want more than one and fewer than the default %d",
			got, trino.DefaultRequestRetryMaxAttempts)
	}
	if !strings.Contains(err.Error(), "request_retry_timeout=500ms") {
		t.Errorf("error %q does not report request_retry_timeout=500ms", err)
	}
}

func TestConfig_DSN_RequestRetry(t *testing.T) {
	base := Config{Host: "localhost", Port: 8080, User: "admin", Source: "test"}

	tests := []struct {
		name         string
		timeout      time.Duration
		maxAttempts  int
		wantTimeout  *time.Duration
		wantAttempts *int
	}{
		{name: "zero leaves the driver defaults"},
		{name: "timeout only", timeout: 5 * time.Second, wantTimeout: ptr(5 * time.Second)},
		{name: "max attempts only", maxAttempts: 1, wantAttempts: ptr(1)},
		{
			name: "both", timeout: 1500 * time.Millisecond, maxAttempts: 4,
			wantTimeout: ptr(1500 * time.Millisecond), wantAttempts: ptr(4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.RequestRetryTimeout = tt.timeout
			cfg.RequestRetryMaxAttempts = tt.maxAttempts

			parsed, err := trino.ParseDSN(cfg.DSN())
			if err != nil {
				t.Fatalf("ParseDSN(%q): %v", cfg.DSN(), err)
			}
			if !equalPtr(parsed.RequestRetryTimeout, tt.wantTimeout) {
				t.Errorf("RequestRetryTimeout = %v, want %v", deref(parsed.RequestRetryTimeout), deref(tt.wantTimeout))
			}
			if !equalPtr(parsed.RequestRetryMaxAttempts, tt.wantAttempts) {
				t.Errorf("RequestRetryMaxAttempts = %v, want %v", deref(parsed.RequestRetryMaxAttempts), deref(tt.wantAttempts))
			}
		})
	}
}

func TestConfig_Validate_RequestRetry(t *testing.T) {
	base := Config{Host: "localhost", Port: 8080, User: "admin"}

	negTimeout := base
	negTimeout.RequestRetryTimeout = -time.Second
	if err := negTimeout.Validate(); err == nil || !strings.Contains(err.Error(), "request retry timeout") {
		t.Errorf("negative RequestRetryTimeout: err = %v, want request retry timeout error", err)
	}

	negAttempts := base
	negAttempts.RequestRetryMaxAttempts = -1
	if err := negAttempts.Validate(); err == nil || !strings.Contains(err.Error(), "request retry max attempts") {
		t.Errorf("negative RequestRetryMaxAttempts: err = %v, want request retry max attempts error", err)
	}

	set := base
	set.RequestRetryTimeout = time.Second
	set.RequestRetryMaxAttempts = 2
	if err := set.Validate(); err != nil {
		t.Errorf("positive retry limits: unexpected error %v", err)
	}
}

func ptr[T any](v T) *T { return &v }

func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref[T any](p *T) any {
	if p == nil {
		return "<unset>"
	}
	return *p
}
