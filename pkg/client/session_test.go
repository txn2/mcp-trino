package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// sessionHeaders is the catalog and schema a query was submitted with. A
// header the driver did not send is absent, which differs from one sent empty.
type sessionHeaders struct {
	catalog, schema       string
	hasCatalog, hasSchema bool
}

// fakeTrino answers each statement with one row, served from a second page
// as Trino does, and records the session headers of each submitted query.
type fakeTrino struct {
	t    *testing.T
	mu   sync.Mutex
	seen []sessionHeaders
}

func (f *fakeTrino) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/v1/statement" {
		catalog, hasCatalog := r.Header[http.CanonicalHeaderKey(trinoCatalogHeader)]
		schema, hasSchema := r.Header[http.CanonicalHeaderKey(trinoSchemaHeader)]
		f.mu.Lock()
		f.seen = append(f.seen, sessionHeaders{
			catalog: strings.Join(catalog, ","), schema: strings.Join(schema, ","),
			hasCatalog: hasCatalog, hasSchema: hasSchema,
		})
		f.mu.Unlock()
	}
	body := `{"id":"q1","stats":{"state":"FINISHED"},` +
		`"columns":[{"name":"c","type":"integer","typeSignature":{"rawType":"integer","arguments":[]}}],` +
		`"data":[[1]]}`
	if r.Method == http.MethodPost {
		body = `{"id":"q1","stats":{"state":"QUEUED"},"nextUri":"http://` + r.Host + `/v1/statement/q1/1"}`
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		f.t.Errorf("write response: %v", err)
	}
}

func (f *fakeTrino) last(t *testing.T) sessionHeaders {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.seen) == 0 {
		t.Fatal("no statement reached the server")
	}
	return f.seen[len(f.seen)-1]
}

// newFakeTrinoClient opens a client through New, so the real driver turns
// cfg's DSN and each query's options into request headers.
func newFakeTrinoClient(t *testing.T, catalog, schema string) (*Client, *fakeTrino) {
	t.Helper()
	fake := &fakeTrino{t: t}
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
		Catalog: catalog, Schema: schema, Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, fake
}

func TestQuery_SessionCatalogSchema(t *testing.T) {
	tests := []struct {
		name            string
		catalog, schema string // Config
		opts            QueryOptions
		want            sessionHeaders
	}{
		{
			name:    "configured catalog and schema",
			catalog: "hive", schema: "sales",
			want: sessionHeaders{catalog: "hive", schema: "sales", hasCatalog: true, hasSchema: true},
		},
		{
			name:    "configured catalog only",
			catalog: "hive",
			want:    sessionHeaders{catalog: "hive", hasCatalog: true},
		},
		{
			name: "neither configured",
			want: sessionHeaders{},
		},
		{
			name:    "per-query catalog and schema",
			catalog: "hive", schema: "sales",
			opts: QueryOptions{Catalog: "iceberg", Schema: "events"},
			want: sessionHeaders{catalog: "iceberg", schema: "events", hasCatalog: true, hasSchema: true},
		},
		{
			name:    "per-query catalog clears configured schema",
			catalog: "hive", schema: "sales",
			opts: QueryOptions{Catalog: "iceberg"},
			want: sessionHeaders{catalog: "iceberg", schema: "", hasCatalog: true, hasSchema: true},
		},
		{
			name:    "per-query schema within configured catalog",
			catalog: "hive", schema: "sales",
			opts: QueryOptions{Schema: "events"},
			want: sessionHeaders{catalog: "hive", schema: "events", hasCatalog: true, hasSchema: true},
		},
		{
			name: "per-query catalog with none configured",
			opts: QueryOptions{Catalog: "iceberg", Schema: "events"},
			want: sessionHeaders{catalog: "iceberg", schema: "events", hasCatalog: true, hasSchema: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, fake := newFakeTrinoClient(t, tt.catalog, tt.schema)
			result, err := c.Query(context.Background(), "SELECT 1", tt.opts)
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			if len(result.Rows) != 1 {
				t.Fatalf("rows = %d, want 1", len(result.Rows))
			}
			if got := fake.last(t); got != tt.want {
				t.Errorf("session headers = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestQuery_SessionPersistsAcrossQueries checks that a per-query override
// does not leak into the pooled connection's next query.
func TestQuery_SessionPersistsAcrossQueries(t *testing.T) {
	c, fake := newFakeTrinoClient(t, "hive", "sales")
	ctx := context.Background()

	if _, err := c.Query(ctx, "SELECT 1", QueryOptions{Catalog: "iceberg", Schema: "events"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Query(ctx, "SELECT 1", QueryOptions{}); err != nil {
		t.Fatal(err)
	}
	want := sessionHeaders{catalog: "hive", schema: "sales", hasCatalog: true, hasSchema: true}
	if got := fake.last(t); got != want {
		t.Errorf("session headers = %+v, want %+v", got, want)
	}
}

func TestQueryStream_SchemaWithoutCatalog(t *testing.T) {
	c, fake := newFakeTrinoClient(t, "", "")
	_, err := c.QueryStream(context.Background(), "SELECT 1", QueryOptions{Schema: "events"})
	if err == nil || !strings.Contains(err.Error(), "needs a catalog") {
		t.Fatalf("err = %v, want a missing-catalog error", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.seen) != 0 {
		t.Errorf("a statement reached the server: %+v", fake.seen)
	}
}
