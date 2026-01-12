package semantic

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheConfig configures the caching behavior.
type CacheConfig struct {
	// TTL is how long cached entries remain valid. Default: 5 minutes.
	TTL time.Duration

	// MaxEntries is the maximum number of entries to cache. Default: 10000.
	// When exceeded, oldest entries are evicted.
	MaxEntries int

	// CacheErrors caches error responses to avoid hammering the provider.
	// Default: false.
	CacheErrors bool

	// ErrorTTL is how long error responses are cached. Default: 1 minute.
	ErrorTTL time.Duration
}

// DefaultCacheConfig returns a CacheConfig with sensible defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		TTL:         5 * time.Minute,
		MaxEntries:  10000,
		CacheErrors: false,
		ErrorTTL:    1 * time.Minute,
	}
}

// cacheEntry holds a cached value with expiration.
type cacheEntry struct {
	value     any
	err       error
	expiresAt time.Time
}

// CachingProvider wraps a Provider with in-memory caching.
type CachingProvider struct {
	provider Provider
	config   CacheConfig

	cache map[string]*cacheEntry
	mu    sync.RWMutex
}

// NewCachingProvider wraps a provider with caching.
func NewCachingProvider(provider Provider, cfg CacheConfig) *CachingProvider {
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 10000
	}
	if cfg.ErrorTTL <= 0 {
		cfg.ErrorTTL = 1 * time.Minute
	}

	return &CachingProvider{
		provider: provider,
		config:   cfg,
		cache:    make(map[string]*cacheEntry),
	}
}

// Name implements Provider.
func (cp *CachingProvider) Name() string {
	return "cached(" + cp.provider.Name() + ")"
}

// GetTableContext implements Provider with caching.
func (cp *CachingProvider) GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error) {
	key := "table:" + table.Key()

	// Check cache
	if entry := cp.get(key); entry != nil {
		if entry.err != nil {
			return nil, entry.err
		}
		if tc, ok := entry.value.(*TableContext); ok {
			return tc, nil
		}
	}

	// Fetch from provider
	result, err := cp.provider.GetTableContext(ctx, table)
	cp.set(key, result, err)
	return result, err
}

// GetColumnContext implements Provider with caching.
func (cp *CachingProvider) GetColumnContext(ctx context.Context, column ColumnIdentifier) (*ColumnContext, error) {
	key := "column:" + column.String()

	if entry := cp.get(key); entry != nil {
		if entry.err != nil {
			return nil, entry.err
		}
		if cc, ok := entry.value.(*ColumnContext); ok {
			return cc, nil
		}
	}

	result, err := cp.provider.GetColumnContext(ctx, column)
	cp.set(key, result, err)
	return result, err
}

// GetColumnsContext implements Provider with caching.
func (cp *CachingProvider) GetColumnsContext(ctx context.Context, table TableIdentifier) (map[string]*ColumnContext, error) {
	key := "columns:" + table.Key()

	if entry := cp.get(key); entry != nil {
		if entry.err != nil {
			return nil, entry.err
		}
		if cols, ok := entry.value.(map[string]*ColumnContext); ok {
			return cols, nil
		}
	}

	result, err := cp.provider.GetColumnsContext(ctx, table)
	cp.set(key, result, err)
	return result, err
}

// GetLineage implements Provider with caching.
func (cp *CachingProvider) GetLineage(
	ctx context.Context, table TableIdentifier, direction LineageDirection, maxDepth int,
) (*LineageInfo, error) {
	key := fmt.Sprintf("lineage:%s:%s:%d", table.Key(), direction, maxDepth)

	if entry := cp.get(key); entry != nil {
		if entry.err != nil {
			return nil, entry.err
		}
		if li, ok := entry.value.(*LineageInfo); ok {
			return li, nil
		}
	}

	result, err := cp.provider.GetLineage(ctx, table, direction, maxDepth)
	cp.set(key, result, err)
	return result, err
}

// GetGlossaryTerm implements Provider with caching.
func (cp *CachingProvider) GetGlossaryTerm(ctx context.Context, urn string) (*GlossaryTerm, error) {
	key := "glossary:" + urn

	if entry := cp.get(key); entry != nil {
		if entry.err != nil {
			return nil, entry.err
		}
		if gt, ok := entry.value.(*GlossaryTerm); ok {
			return gt, nil
		}
	}

	result, err := cp.provider.GetGlossaryTerm(ctx, urn)
	cp.set(key, result, err)
	return result, err
}

// SearchTables implements Provider (not cached due to variability).
func (cp *CachingProvider) SearchTables(ctx context.Context, filter SearchFilter) ([]TableIdentifier, error) {
	// Search is not cached as results vary by filter
	return cp.provider.SearchTables(ctx, filter)
}

// Close implements Provider.
func (cp *CachingProvider) Close() error {
	cp.mu.Lock()
	cp.cache = make(map[string]*cacheEntry)
	cp.mu.Unlock()
	return cp.provider.Close()
}

// Stats returns cache statistics.
func (cp *CachingProvider) Stats() CacheStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	now := time.Now()
	active := 0
	expired := 0
	for _, entry := range cp.cache {
		if now.Before(entry.expiresAt) {
			active++
		} else {
			expired++
		}
	}

	return CacheStats{
		TotalEntries:   len(cp.cache),
		ActiveEntries:  active,
		ExpiredEntries: expired,
		MaxEntries:     cp.config.MaxEntries,
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalEntries   int
	ActiveEntries  int
	ExpiredEntries int
	MaxEntries     int
}

// Clear removes all entries from the cache.
func (cp *CachingProvider) Clear() {
	cp.mu.Lock()
	cp.cache = make(map[string]*cacheEntry)
	cp.mu.Unlock()
}

// get retrieves a non-expired entry from cache.
func (cp *CachingProvider) get(key string) *cacheEntry {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	entry, ok := cp.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry
}

// set adds an entry to the cache.
func (cp *CachingProvider) set(key string, value any, err error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check if we should cache errors
	if err != nil && !cp.config.CacheErrors {
		return
	}

	// Determine TTL
	ttl := cp.config.TTL
	if err != nil {
		ttl = cp.config.ErrorTTL
	}

	// Evict if at capacity
	if len(cp.cache) >= cp.config.MaxEntries {
		cp.evictOldest()
	}

	cp.cache[key] = &cacheEntry{
		value:     value,
		err:       err,
		expiresAt: time.Now().Add(ttl),
	}
}

// evictOldest removes the oldest entry from the cache.
func (cp *CachingProvider) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range cp.cache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(cp.cache, oldestKey)
	}
}

// Verify CachingProvider implements Provider.
var _ Provider = (*CachingProvider)(nil)
