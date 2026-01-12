# Caching

The semantic layer includes an in-memory cache to minimize latency and reduce load on metadata catalogs. Caching wraps any provider transparently.

## Configuration

```go
import (
    "github.com/txn2/mcp-trino/pkg/semantic"
    "github.com/txn2/mcp-trino/pkg/tools"
)

toolkit := tools.NewToolkit(trinoClient, cfg,
    tools.WithSemanticProvider(datahubProvider),
    tools.WithSemanticCache(semantic.CacheConfig{
        TTL:         5 * time.Minute,
        MaxEntries:  10000,
        CacheErrors: false,
        ErrorTTL:    1 * time.Minute,
    }),
)
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `TTL` | 5 minutes | How long cached entries remain valid |
| `MaxEntries` | 10,000 | Maximum cache size (LRU eviction when exceeded) |
| `CacheErrors` | false | Whether to cache error responses |
| `ErrorTTL` | 1 minute | TTL for cached errors (if enabled) |

### Default Configuration

```go
cfg := semantic.DefaultCacheConfig()
// TTL: 5 minutes
// MaxEntries: 10000
// CacheErrors: false
// ErrorTTL: 1 minute
```

## Cached Operations

| Method | Cached |
|--------|--------|
| `GetTableContext` | Yes |
| `GetColumnContext` | Yes |
| `GetColumnsContext` | Yes |
| `GetLineage` | Yes |
| `GetGlossaryTerm` | Yes |
| `SearchTables` | No (results vary by filter) |

## Cache Keys

Cache keys are derived from the operation and identifier:

- `table:{catalog}.{schema}.{table}`
- `column:{catalog}.{schema}.{table}.{column}`
- `columns:{catalog}.{schema}.{table}`
- `lineage:{catalog}.{schema}.{table}:{direction}:{maxDepth}`
- `glossary:{urn}`

## Manual Cache Control

The `CachingProvider` exposes methods for cache management:

```go
import "github.com/txn2/mcp-trino/pkg/semantic"

// Wrap provider with caching
cached := semantic.NewCachingProvider(provider, semantic.DefaultCacheConfig())

// Get cache statistics
stats := cached.Stats()
fmt.Printf("Active: %d, Expired: %d, Max: %d\n",
    stats.ActiveEntries, stats.ExpiredEntries, stats.MaxEntries)

// Clear all entries
cached.Clear()
```

## Error Caching

By default, errors are not cached. Enable error caching to prevent hammering a failing provider:

```go
cfg := semantic.CacheConfig{
    TTL:         5 * time.Minute,
    MaxEntries:  10000,
    CacheErrors: true,   // Cache errors
    ErrorTTL:    1 * time.Minute,  // Shorter TTL for errors
}
```

When an error is cached:

- Subsequent requests return the cached error immediately
- The error expires after `ErrorTTL`
- After expiration, the next request retries the provider

## Eviction Policy

When `MaxEntries` is reached, the oldest entry (by expiration time) is evicted to make room for new entries. This approximates LRU behavior.

## Disabling Cache

To disable caching, omit the `WithSemanticCache` option:

```go
toolkit := tools.NewToolkit(trinoClient, cfg,
    tools.WithSemanticProvider(datahubProvider),
    // No WithSemanticCache - caching disabled
)
```

## Performance Considerations

### Sizing

Choose `MaxEntries` based on your table count:

| Tables | Suggested MaxEntries |
|--------|---------------------|
| < 1,000 | 10,000 (default) |
| 1,000 - 10,000 | 50,000 |
| > 10,000 | 100,000+ |

Each entry requires approximately 1-5 KB depending on metadata richness.

### TTL Tradeoffs

| TTL | Behavior |
|-----|----------|
| Short (1m) | Fresh data, more catalog load |
| Medium (5m) | Balanced (default) |
| Long (30m+) | Lower load, stale data risk |

For catalogs with infrequent metadata changes, longer TTLs are safe. For actively evolving metadata, prefer shorter TTLs.

## Monitoring

Track cache effectiveness by checking stats periodically:

```go
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        stats := cached.Stats()
        log.Printf("Cache: active=%d expired=%d max=%d",
            stats.ActiveEntries, stats.ExpiredEntries, stats.MaxEntries)
    }
}()
```

High expired counts relative to active entries may indicate the TTL is too short.
