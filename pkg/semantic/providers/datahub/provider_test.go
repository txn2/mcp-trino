//nolint:errcheck // test file intentionally ignores some return values
package datahub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/txn2/mcp-trino/pkg/semantic"
)

// mockGraphQLServer creates a mock DataHub GraphQL server.
func mockGraphQLServer(t *testing.T, handler func(query string, variables map[string]any) (any, error)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data, err := handler(req.Query, req.Variables)
		if err != nil {
			resp := map[string]any{
				"errors": []map[string]string{{"message": err.Error()}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		resp := map[string]any{
			"data": data,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestNew(t *testing.T) {
	t.Run("success with valid config", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return nil, nil
		})
		defer server.Close()

		p, err := New(Config{
			Endpoint: server.URL,
			Token:    "test-token",
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if p == nil {
			t.Error("New() returned nil")
		}
	})

	t.Run("error on missing endpoint", func(t *testing.T) {
		_, err := New(Config{Token: "test-token"})
		if !errors.Is(err, ErrNoEndpoint) {
			t.Errorf("New() error = %v, want ErrNoEndpoint", err)
		}
	})

	t.Run("error on missing token", func(t *testing.T) {
		_, err := New(Config{Endpoint: "http://localhost"})
		if !errors.Is(err, ErrNoToken) {
			t.Errorf("New() error = %v, want ErrNoToken", err)
		}
	})
}

func TestProvider_Name(t *testing.T) {
	server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
		return nil, nil
	})
	defer server.Close()

	p, _ := New(Config{Endpoint: server.URL, Token: "test"})
	if got := p.Name(); got != ProviderName {
		t.Errorf("Name() = %q, want %q", got, ProviderName)
	}
}

func TestProvider_GetTableContext(t *testing.T) {
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	ctx := context.Background()

	t.Run("returns table context", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"dataset": map[string]any{
					"urn": "urn:li:dataset:test",
					"properties": map[string]any{
						"name":        "users",
						"description": "User table",
					},
					"ownership": map[string]any{
						"owners": []map[string]any{
							{
								"owner": map[string]any{
									"urn":      "urn:li:corpuser:alice",
									"username": "alice",
									"properties": map[string]any{
										"displayName": "Alice Smith",
									},
								},
								"ownershipType": map[string]any{
									"urn": "urn:li:ownershipType:technical",
									"info": map[string]any{
										"name": "Technical Owner",
									},
								},
							},
						},
					},
					"tags": map[string]any{
						"tags": []map[string]any{
							{
								"tag": map[string]any{
									"urn": "urn:li:tag:pii",
									"properties": map[string]any{
										"name":        "pii",
										"description": "Contains PII",
									},
								},
							},
						},
					},
					"glossaryTerms": map[string]any{
						"terms": []map[string]any{
							{
								"term": map[string]any{
									"urn": "urn:li:glossaryTerm:customer",
									"properties": map[string]any{
										"name":       "Customer",
										"definition": "A customer",
									},
								},
							},
						},
					},
					"domain": map[string]any{
						"domain": map[string]any{
							"urn": "urn:li:domain:customer",
							"properties": map[string]any{
								"name":        "Customer",
								"description": "Customer domain",
							},
						},
					},
					"deprecation": map[string]any{
						"deprecated":       true,
						"note":             "Use v2",
						"decommissionTime": time.Now().Add(24 * time.Hour).UnixMilli(),
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetTableContext(ctx, table)
		if err != nil {
			t.Fatalf("GetTableContext() error = %v", err)
		}
		if result == nil {
			t.Fatal("GetTableContext() returned nil")
		}
		if result.Description != "User table" {
			t.Errorf("Description = %q", result.Description)
		}
		if result.Source != ProviderName {
			t.Errorf("Source = %q", result.Source)
		}
		if result.Ownership == nil || len(result.Ownership.Owners) != 1 {
			t.Error("Ownership not set correctly")
		}
		if len(result.Tags) != 1 {
			t.Errorf("Tags count = %d", len(result.Tags))
		}
		if len(result.GlossaryTerms) != 1 {
			t.Errorf("GlossaryTerms count = %d", len(result.GlossaryTerms))
		}
		if result.Domain == nil {
			t.Error("Domain is nil")
		}
		if result.Deprecation == nil {
			t.Error("Deprecation is nil")
		}
	})

	t.Run("returns nil for unknown table", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{"dataset": nil}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("GetTableContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetTableContext() = %v, want nil", result)
		}
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		_, err := p.GetTableContext(ctx, table)
		if err == nil {
			t.Error("GetTableContext() expected error")
		}
	})
}

func TestProvider_GetColumnContext(t *testing.T) {
	column := semantic.ColumnIdentifier{
		TableIdentifier: semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
		Column:          "user_id",
	}
	ctx := context.Background()

	t.Run("returns column context", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"dataset": map[string]any{
					"schemaMetadata": map[string]any{
						"fields": []map[string]any{
							{
								"fieldPath":   "user_id",
								"description": "User identifier",
								"tags": map[string]any{
									"tags": []map[string]any{
										{
											"tag": map[string]any{
												"urn": "urn:li:tag:pii",
												"properties": map[string]any{
													"name": "pii",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			t.Fatalf("GetColumnContext() error = %v", err)
		}
		if result == nil {
			t.Fatal("GetColumnContext() returned nil")
		}
		if result.Description != "User identifier" {
			t.Errorf("Description = %q", result.Description)
		}
		if !result.IsSensitive {
			t.Error("IsSensitive should be true for pii tag")
		}
	})

	t.Run("returns nil for unknown column", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"dataset": map[string]any{
					"schemaMetadata": map[string]any{
						"fields": []map[string]any{
							{"fieldPath": "other_column", "description": "Other"},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnContext() = %v, want nil", result)
		}
	})

	t.Run("returns nil for table without schema", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{"dataset": nil}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			t.Errorf("GetColumnContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnContext() = %v, want nil", result)
		}
	})
}

func TestProvider_GetColumnsContext(t *testing.T) {
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	ctx := context.Background()

	t.Run("returns all columns", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"dataset": map[string]any{
					"schemaMetadata": map[string]any{
						"fields": []map[string]any{
							{"fieldPath": "user_id", "description": "User ID"},
							{"fieldPath": "parent.email", "description": "Email"},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetColumnsContext(ctx, table)
		if err != nil {
			t.Fatalf("GetColumnsContext() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
		if result["user_id"] == nil {
			t.Error("user_id missing")
		}
		if result["email"] == nil {
			t.Error("email missing (should extract from parent.email)")
		}
	})

	t.Run("returns nil for empty schema", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"dataset": map[string]any{
					"schemaMetadata": nil,
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetColumnsContext(ctx, table)
		if err != nil {
			t.Errorf("GetColumnsContext() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetColumnsContext() = %v, want nil", result)
		}
	})
}

func TestProvider_GetLineage(t *testing.T) {
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	ctx := context.Background()

	t.Run("returns upstream lineage", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"searchAcrossLineage": map[string]any{
					"searchResults": []map[string]any{
						{
							"degree": 1,
							"entity": map[string]any{
								"urn":  "urn:li:dataset:(urn:li:dataPlatform:trino,hive.raw.users_raw,PROD)",
								"type": "DATASET",
								"name": "users_raw",
							},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetLineage(ctx, table, semantic.LineageUpstream, 3)
		if err != nil {
			t.Fatalf("GetLineage() error = %v", err)
		}
		if result == nil {
			t.Fatal("GetLineage() returned nil")
		}
		if result.Direction != semantic.LineageUpstream {
			t.Errorf("Direction = %s", result.Direction)
		}
		if len(result.Edges) != 1 {
			t.Errorf("Edges count = %d, want 1", len(result.Edges))
		}
		if result.Edges[0].SourceTable.Table != "users_raw" {
			t.Errorf("SourceTable = %s", result.Edges[0].SourceTable.Table)
		}
	})

	t.Run("returns downstream lineage", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, vars map[string]any) (any, error) {
			if vars["direction"] != "DOWNSTREAM" {
				t.Error("expected DOWNSTREAM direction")
			}
			return map[string]any{
				"searchAcrossLineage": map[string]any{
					"searchResults": []map[string]any{
						{
							"entity": map[string]any{
								"urn":  "urn:li:dataset:(urn:li:dataPlatform:trino,hive.marts.users_summary,PROD)",
								"type": "DATASET",
							},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetLineage(ctx, table, semantic.LineageDownstream, 3)
		if err != nil {
			t.Fatalf("GetLineage() error = %v", err)
		}
		if result.Direction != semantic.LineageDownstream {
			t.Errorf("Direction = %s", result.Direction)
		}
	})

	t.Run("returns nil for no lineage", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"searchAcrossLineage": map[string]any{
					"searchResults": []map[string]any{},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetLineage(ctx, table, semantic.LineageUpstream, 3)
		if err != nil {
			t.Errorf("GetLineage() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetLineage() = %v, want nil", result)
		}
	})

	t.Run("skips non-dataset entities", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"searchAcrossLineage": map[string]any{
					"searchResults": []map[string]any{
						{
							"entity": map[string]any{
								"urn":  "urn:li:dataJob:something",
								"type": "DATA_JOB",
							},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetLineage(ctx, table, semantic.LineageUpstream, 3)
		if err != nil {
			t.Fatalf("GetLineage() error = %v", err)
		}
		if len(result.Edges) != 0 {
			t.Errorf("expected 0 edges, got %d", len(result.Edges))
		}
	})
}

func TestProvider_GetGlossaryTerm(t *testing.T) {
	ctx := context.Background()
	urn := "urn:li:glossaryTerm:customer"

	t.Run("returns glossary term", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"glossaryTerm": map[string]any{
					"urn": urn,
					"properties": map[string]any{
						"name":       "Customer",
						"definition": "A person who buys things",
						"termSource": "EXTERNAL",
					},
					"relatedTerms": map[string]any{
						"terms": []map[string]any{
							{
								"term": map[string]any{
									"urn": "urn:li:glossaryTerm:user",
								},
							},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetGlossaryTerm(ctx, urn)
		if err != nil {
			t.Fatalf("GetGlossaryTerm() error = %v", err)
		}
		if result == nil {
			t.Fatal("GetGlossaryTerm() returned nil")
		}
		if result.Name != "Customer" {
			t.Errorf("Name = %q", result.Name)
		}
		if result.Definition != "A person who buys things" {
			t.Errorf("Definition = %q", result.Definition)
		}
		if len(result.RelatedTerms) != 1 {
			t.Errorf("RelatedTerms count = %d", len(result.RelatedTerms))
		}
	})

	t.Run("returns nil for unknown term", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{"glossaryTerm": nil}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.GetGlossaryTerm(ctx, urn)
		if err != nil {
			t.Errorf("GetGlossaryTerm() error = %v", err)
		}
		if result != nil {
			t.Errorf("GetGlossaryTerm() = %v, want nil", result)
		}
	})
}

func TestProvider_SearchTables(t *testing.T) {
	ctx := context.Background()

	t.Run("returns search results", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"search": map[string]any{
					"searchResults": []map[string]any{
						{
							"entity": map[string]any{
								"urn": "urn:li:dataset:(urn:li:dataPlatform:trino,hive.analytics.users,PROD)",
							},
						},
						{
							"entity": map[string]any{
								"urn": "urn:li:dataset:(urn:li:dataPlatform:trino,hive.analytics.orders,PROD)",
							},
						},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.SearchTables(ctx, semantic.SearchFilter{Query: "test"})
		if err != nil {
			t.Fatalf("SearchTables() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
	})

	t.Run("uses default query", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, vars map[string]any) (any, error) {
			if vars["query"] != "*" {
				t.Errorf("query = %q, want '*'", vars["query"])
			}
			return map[string]any{
				"search": map[string]any{
					"searchResults": []map[string]any{},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		_, _ = p.SearchTables(ctx, semantic.SearchFilter{})
	})

	t.Run("applies limit", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, vars map[string]any) (any, error) {
			if count, ok := vars["count"].(float64); !ok || int(count) != 10 {
				t.Errorf("count = %v, want 10", vars["count"])
			}
			return map[string]any{
				"search": map[string]any{"searchResults": []map[string]any{}},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		_, _ = p.SearchTables(ctx, semantic.SearchFilter{Limit: 10})
	})

	t.Run("adds domain filter", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, vars map[string]any) (any, error) {
			// JSON unmarshaling creates []interface{} not []map[string]any
			filters, ok := vars["filters"].([]interface{})
			if !ok || len(filters) == 0 {
				t.Error("expected domain filter")
			}
			return map[string]any{
				"search": map[string]any{"searchResults": []map[string]any{}},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		_, _ = p.SearchTables(ctx, semantic.SearchFilter{Domain: "customer"})
	})

	t.Run("adds tag filter", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, vars map[string]any) (any, error) {
			// JSON unmarshaling creates []interface{} not []map[string]any
			filters, ok := vars["filters"].([]interface{})
			if !ok || len(filters) == 0 {
				t.Error("expected tag filter")
			}
			return map[string]any{
				"search": map[string]any{"searchResults": []map[string]any{}},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		_, _ = p.SearchTables(ctx, semantic.SearchFilter{Tags: []string{"pii"}})
	})

	t.Run("skips invalid URNs", func(t *testing.T) {
		server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
			return map[string]any{
				"search": map[string]any{
					"searchResults": []map[string]any{
						{"entity": map[string]any{"urn": "invalid-urn"}},
						{"entity": map[string]any{"urn": "urn:li:dataset:(urn:li:dataPlatform:trino,hive.analytics.valid,PROD)"}},
					},
				},
			}, nil
		})
		defer server.Close()

		p, _ := New(Config{Endpoint: server.URL, Token: "test"})
		result, err := p.SearchTables(ctx, semantic.SearchFilter{})
		if err != nil {
			t.Fatalf("SearchTables() error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1 (invalid URN skipped)", len(result))
		}
	})
}

func TestProvider_Close(t *testing.T) {
	server := mockGraphQLServer(t, func(_ string, _ map[string]any) (any, error) {
		return nil, nil
	})
	defer server.Close()

	p, _ := New(Config{Endpoint: server.URL, Token: "test"})
	err := p.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := Config{Endpoint: "http://localhost", Token: "test"}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v", err)
		}
	})

	t.Run("missing endpoint", func(t *testing.T) {
		cfg := Config{Token: "test"}
		if err := cfg.Validate(); !errors.Is(err, ErrNoEndpoint) {
			t.Errorf("Validate() error = %v, want ErrNoEndpoint", err)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		cfg := Config{Endpoint: "http://localhost"}
		if err := cfg.Validate(); !errors.Is(err, ErrNoToken) {
			t.Errorf("Validate() error = %v, want ErrNoToken", err)
		}
	})
}

func TestConfig_WithDefaults(t *testing.T) {
	cfg := Config{Endpoint: "http://localhost", Token: "test"}
	cfg = cfg.WithDefaults()

	if cfg.Platform != "trino" {
		t.Errorf("Platform = %q, want 'trino'", cfg.Platform)
	}
	if cfg.Environment != "PROD" {
		t.Errorf("Environment = %q, want 'PROD'", cfg.Environment)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestFromEnv(t *testing.T) {
	// Save and restore env
	envVars := []string{"DATAHUB_ENDPOINT", "DATAHUB_TOKEN", "DATAHUB_PLATFORM", "DATAHUB_ENVIRONMENT", "DATAHUB_TIMEOUT"}
	saved := make(map[string]string)
	for _, key := range envVars {
		saved[key] = os.Getenv(key)
	}
	defer func() {
		for key, val := range saved {
			_ = os.Setenv(key, val)
		}
	}()

	// Clear all env vars first
	for _, key := range envVars {
		_ = os.Unsetenv(key)
	}

	t.Run("reads all env vars", func(t *testing.T) {
		t.Setenv("DATAHUB_ENDPOINT", "http://test")
		t.Setenv("DATAHUB_TOKEN", "token123")
		t.Setenv("DATAHUB_PLATFORM", "custom")
		t.Setenv("DATAHUB_ENVIRONMENT", "DEV")
		t.Setenv("DATAHUB_TIMEOUT", "60s")

		cfg := FromEnv()
		if cfg.Endpoint != "http://test" {
			t.Errorf("Endpoint = %q", cfg.Endpoint)
		}
		if cfg.Token != "token123" {
			t.Errorf("Token = %q", cfg.Token)
		}
		if cfg.Platform != "custom" {
			t.Errorf("Platform = %q", cfg.Platform)
		}
		if cfg.Environment != "DEV" {
			t.Errorf("Environment = %q", cfg.Environment)
		}
		if cfg.Timeout != 60*time.Second {
			t.Errorf("Timeout = %v", cfg.Timeout)
		}
	})

	t.Run("uses defaults", func(t *testing.T) {
		for _, key := range envVars {
			_ = os.Unsetenv(key)
		}

		cfg := FromEnv()
		if cfg.Platform != "trino" {
			t.Errorf("Platform = %q, want 'trino'", cfg.Platform)
		}
		if cfg.Environment != "PROD" {
			t.Errorf("Environment = %q, want 'PROD'", cfg.Environment)
		}
		if cfg.Timeout != 30*time.Second {
			t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
		}
	})

	t.Run("ignores invalid timeout", func(t *testing.T) {
		t.Setenv("DATAHUB_TIMEOUT", "invalid")
		cfg := FromEnv()
		if cfg.Timeout != 30*time.Second {
			t.Errorf("Timeout = %v, want 30s for invalid input", cfg.Timeout)
		}
	})
}

func TestBuildDatasetURN(t *testing.T) {
	table := semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"}
	urn := buildDatasetURN(table, "trino", "PROD")
	expected := "urn:li:dataset:(urn:li:dataPlatform:trino,hive.analytics.users,PROD)"
	if urn != expected {
		t.Errorf("buildDatasetURN() = %q, want %q", urn, expected)
	}
}

func TestParseDatasetURN(t *testing.T) {
	tests := []struct {
		name    string
		urn     string
		want    *semantic.TableIdentifier
		wantErr bool
	}{
		{
			name: "valid URN",
			urn:  "urn:li:dataset:(urn:li:dataPlatform:trino,hive.analytics.users,PROD)",
			want: &semantic.TableIdentifier{Catalog: "hive", Schema: "analytics", Table: "users"},
		},
		{
			name:    "invalid prefix",
			urn:     "invalid-urn",
			wantErr: true,
		},
		{
			name:    "missing parentheses",
			urn:     "urn:li:dataset:no-parens",
			wantErr: true,
		},
		{
			name:    "invalid qualified name",
			urn:     "urn:li:dataset:(urn:li:dataPlatform:trino,invalid,PROD)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDatasetURN(tt.urn)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDatasetURN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				if got == nil {
					t.Fatal("parseDatasetURN() returned nil")
				}
				if *got != *tt.want {
					t.Errorf("parseDatasetURN() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMapOwnerEntry(t *testing.T) {
	t.Run("maps user owner", func(t *testing.T) {
		entry := ownerEntry{
			Owner: ownerInfo{
				URN:      "urn:li:corpuser:alice",
				Username: "alice",
				Properties: &struct {
					DisplayName string `json:"displayName"`
					Email       string `json:"email"`
				}{DisplayName: "Alice Smith"},
			},
			OwnershipType: &struct {
				URN  string `json:"urn"`
				Info *struct {
					Name string `json:"name"`
				} `json:"info"`
			}{
				Info: &struct {
					Name string `json:"name"`
				}{Name: "Technical Owner"},
			},
		}

		result := mapOwnerEntry(entry)
		if result.Type != "user" {
			t.Errorf("Type = %q, want 'user'", result.Type)
		}
		if result.Name != "Alice Smith" {
			t.Errorf("Name = %q, want 'Alice Smith'", result.Name)
		}
		if result.Role != "Technical Owner" {
			t.Errorf("Role = %q", result.Role)
		}
	})

	t.Run("maps group owner", func(t *testing.T) {
		entry := ownerEntry{
			Owner: ownerInfo{
				URN:  "urn:li:corpGroup:data-team",
				Name: "data-team",
			},
		}

		result := mapOwnerEntry(entry)
		if result.Type != "group" {
			t.Errorf("Type = %q, want 'group'", result.Type)
		}
		if result.Name != "data-team" {
			t.Errorf("Name = %q", result.Name)
		}
	})
}

func TestExtractColumnName(t *testing.T) {
	tests := []struct {
		fieldPath string
		expected  string
	}{
		{"column_name", "column_name"},
		{"parent.child", "child"},
		{"a.b.c", "c"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldPath, func(t *testing.T) {
			got := extractColumnName(tt.fieldPath)
			if got != tt.expected {
				t.Errorf("extractColumnName(%q) = %q, want %q", tt.fieldPath, got, tt.expected)
			}
		})
	}
}

func TestMapColumnTagsWithSensitivity(t *testing.T) {
	t.Run("detects pii tag", func(t *testing.T) {
		data := &tagsData{
			Tags: []tagEntry{
				{Tag: struct {
					URN        string `json:"urn"`
					Properties *struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					} `json:"properties"`
				}{
					Properties: &struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					}{Name: "PII"},
				}},
			},
		}

		tags, isSensitive, level := mapColumnTagsWithSensitivity(data)
		if !isSensitive {
			t.Error("expected isSensitive to be true")
		}
		if level != "PII" {
			t.Errorf("sensitivityLevel = %q", level)
		}
		if len(tags) != 1 {
			t.Errorf("tags count = %d", len(tags))
		}
	})

	t.Run("detects sensitive tag", func(t *testing.T) {
		data := &tagsData{
			Tags: []tagEntry{
				{Tag: struct {
					URN        string `json:"urn"`
					Properties *struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					} `json:"properties"`
				}{
					Properties: &struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					}{Name: "highly-sensitive"},
				}},
			},
		}

		_, isSensitive, _ := mapColumnTagsWithSensitivity(data)
		if !isSensitive {
			t.Error("expected isSensitive to be true for 'sensitive' tag")
		}
	})

	t.Run("nil data returns no sensitivity", func(t *testing.T) {
		tags, isSensitive, level := mapColumnTagsWithSensitivity(nil)
		if tags != nil {
			t.Errorf("tags = %v, want nil", tags)
		}
		if isSensitive {
			t.Error("isSensitive should be false")
		}
		if level != "" {
			t.Errorf("sensitivityLevel = %q, want empty", level)
		}
	})
}

func TestProvider_ImplementsInterface(_ *testing.T) {
	var _ semantic.Provider = (*Provider)(nil)
}
