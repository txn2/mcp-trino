# Custom Providers

Build custom providers to integrate mcp-trino with any metadata catalog, including Atlan, OpenMetadata, Alation, dbt docs, or internal systems.

## Provider Interface

Implement the `semantic.Provider` interface:

```go
type Provider interface {
    // Name returns the provider name for logging and attribution
    Name() string

    // GetTableContext retrieves metadata for a table
    // Returns nil (not error) if not found
    GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error)

    // GetColumnContext retrieves metadata for a column
    // Returns nil (not error) if not found
    GetColumnContext(ctx context.Context, column ColumnIdentifier) (*ColumnContext, error)

    // GetColumnsContext retrieves metadata for all columns in a table
    // Returns empty map (not error) if not found
    GetColumnsContext(ctx context.Context, table TableIdentifier) (map[string]*ColumnContext, error)

    // GetLineage retrieves upstream or downstream lineage
    // Returns nil (not error) if not found
    GetLineage(ctx context.Context, table TableIdentifier, direction LineageDirection, maxDepth int) (*LineageInfo, error)

    // GetGlossaryTerm retrieves a glossary term by URN
    // Returns nil (not error) if not found
    GetGlossaryTerm(ctx context.Context, urn string) (*GlossaryTerm, error)

    // SearchTables finds tables matching the filter
    // Returns empty slice (not error) if no matches
    SearchTables(ctx context.Context, filter SearchFilter) ([]TableIdentifier, error)

    // Close releases resources
    Close() error
}
```

## Error Handling Convention

The interface distinguishes between "not found" and "error":

- **Not found**: Return `nil` for the result, `nil` for the error
- **Error**: Return `nil` for the result, non-nil error

This ensures missing metadata doesn't break tool execution.

```go
func (p *MyProvider) GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error) {
    result, err := p.catalog.Lookup(table)
    if err != nil {
        if errors.Is(err, catalog.ErrNotFound) {
            return nil, nil  // Not found - not an error
        }
        return nil, err  // Actual error (connection, auth, etc.)
    }
    return mapToTableContext(result), nil
}
```

## Example: Custom Provider

```go
package myprovider

import (
    "context"
    "github.com/txn2/mcp-trino/pkg/semantic"
)

type MyProvider struct {
    client *mycatalog.Client
}

func New(endpoint, token string) (*MyProvider, error) {
    client, err := mycatalog.NewClient(endpoint, token)
    if err != nil {
        return nil, err
    }
    return &MyProvider{client: client}, nil
}

func (p *MyProvider) Name() string {
    return "mycatalog"
}

func (p *MyProvider) GetTableContext(ctx context.Context, table semantic.TableIdentifier) (*semantic.TableContext, error) {
    // Query your catalog
    asset, err := p.client.GetDataset(ctx, table.Catalog, table.Schema, table.Table)
    if err != nil {
        if errors.Is(err, mycatalog.ErrNotFound) {
            return nil, nil
        }
        return nil, err
    }

    // Map to semantic types
    return &semantic.TableContext{
        Identifier:  table,
        Description: asset.Description,
        Ownership: &semantic.Ownership{
            Owners: []semantic.Owner{
                {Name: asset.Owner, Type: "user", Role: "Data Owner"},
            },
        },
        Tags:   mapTags(asset.Tags),
        Source: "mycatalog",
    }, nil
}

func (p *MyProvider) GetColumnsContext(ctx context.Context, table semantic.TableIdentifier) (map[string]*semantic.ColumnContext, error) {
    // Implement column metadata retrieval
    return nil, nil
}

func (p *MyProvider) GetColumnContext(ctx context.Context, column semantic.ColumnIdentifier) (*semantic.ColumnContext, error) {
    return nil, nil
}

func (p *MyProvider) GetLineage(ctx context.Context, table semantic.TableIdentifier, direction semantic.LineageDirection, maxDepth int) (*semantic.LineageInfo, error) {
    return nil, nil
}

func (p *MyProvider) GetGlossaryTerm(ctx context.Context, urn string) (*semantic.GlossaryTerm, error) {
    return nil, nil
}

func (p *MyProvider) SearchTables(ctx context.Context, filter semantic.SearchFilter) ([]semantic.TableIdentifier, error) {
    return nil, nil
}

func (p *MyProvider) Close() error {
    return p.client.Close()
}
```

## ProviderFunc: Function-Based Providers

For simple providers that don't need struct state, use `ProviderFunc`:

```go
import "github.com/txn2/mcp-trino/pkg/semantic"

provider := semantic.ProviderFunc{
    NameFn: func() string { return "inline" },

    GetTableContextFn: func(ctx context.Context, table semantic.TableIdentifier) (*semantic.TableContext, error) {
        if table.Table == "customers" {
            return &semantic.TableContext{
                Identifier:  table,
                Description: "Customer master data",
            }, nil
        }
        return nil, nil
    },
}
```

Each function is optional. Unset functions return nil.

## ProviderChain: Composing Providers

Chain multiple providers for fallback or layering:

```go
import "github.com/txn2/mcp-trino/pkg/semantic"

chain := semantic.NewProviderChain(
    localOverrides,   // Check local overrides first
    datahubProvider,  // Fall back to DataHub
)

toolkit := tools.NewToolkit(client, cfg,
    tools.WithSemanticProvider(chain),
)
```

### Chain Behavior

| Method | Behavior |
|--------|----------|
| `GetTableContext` | First non-nil result wins |
| `GetColumnContext` | First non-nil result wins |
| `GetColumnsContext` | Merges results (later providers override) |
| `GetLineage` | First non-nil result wins |
| `GetGlossaryTerm` | First non-nil result wins |
| `SearchTables` | Combines results with deduplication |

### Override Pattern

Use chaining to add local overrides on top of a catalog:

```go
// Local overrides for specific tables
overrides := semantic.ProviderFunc{
    NameFn: func() string { return "overrides" },
    GetTableContextFn: func(ctx context.Context, t semantic.TableIdentifier) (*semantic.TableContext, error) {
        if t.Table == "customers" {
            return &semantic.TableContext{
                Identifier:  t,
                Description: "OVERRIDE: Use v2_customers instead",
                Deprecation: &semantic.Deprecation{
                    Deprecated: true,
                    ReplacedBy: "v2_customers",
                },
            }, nil
        }
        return nil, nil  // Fall through to next provider
    },
}

chain := semantic.NewProviderChain(overrides, datahubProvider)
```

## Thread Safety

Providers must be safe for concurrent use. Multiple goroutines may call provider methods simultaneously.

## Testing

Test providers using the standard interface:

```go
func TestMyProvider(t *testing.T) {
    provider, err := myprovider.New("http://test", "token")
    require.NoError(t, err)
    defer provider.Close()

    ctx := context.Background()
    table := semantic.TableIdentifier{
        Catalog: "hive",
        Schema:  "analytics",
        Table:   "customers",
    }

    // Test table context
    tc, err := provider.GetTableContext(ctx, table)
    require.NoError(t, err)
    require.NotNil(t, tc)
    assert.Equal(t, "Customer master data", tc.Description)

    // Test not found returns nil, not error
    notFound := semantic.TableIdentifier{Catalog: "x", Schema: "y", Table: "z"}
    tc, err = provider.GetTableContext(ctx, notFound)
    require.NoError(t, err)
    assert.Nil(t, tc)
}
```

## Registering with Toolkit

```go
toolkit := tools.NewToolkit(trinoClient, cfg,
    tools.WithSemanticProvider(myProvider),
    tools.WithSemanticCache(semantic.DefaultCacheConfig()),  // Optional caching
)
```
