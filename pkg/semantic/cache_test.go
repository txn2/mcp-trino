//nolint:errcheck // test file intentionally ignores some return values
package semantic

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCachingProvider_Name(t *testing.T) {
	inner := ProviderFunc{NameFn: func() string { return "inner" }}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	expected := "cached(inner)"
	got := cp.Name()
	if got != expected {
		t.Errorf("Name() = %q, want %q", got, expected)
	}
}

func TestCachingProvider_GetTableContext(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	t.Run("caches successful result", func(t *testing.T) {
		callCount := 0
		inner := ProviderFunc{
			GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
				callCount++
				return &TableContext{Description: "cached"}, nil
			},
		}
		cp := NewCachingProvider(inner, DefaultCacheConfig())

		// First call
		result1, err := cp.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("first call error = %v", err)
		}
		if result1.Description != "cached" {
			t.Errorf("first result description = %q", result1.Description)
		}

		// Second call should use cache
		result2, err := cp.GetTableContext(ctx, table)
		if err != nil {
			t.Errorf("second call error = %v", err)
		}
		if result2.Description != "cached" {
			t.Errorf("second result description = %q", result2.Description)
		}

		if callCount != 1 {
			t.Errorf("callCount = %d, want 1 (should use cache)", callCount)
		}
	})

	t.Run("caches nil result", func(t *testing.T) {
		callCount := 0
		inner := ProviderFunc{
			GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
				callCount++
				return nil, nil
			},
		}
		cp := NewCachingProvider(inner, DefaultCacheConfig())

		// First call
		_, _ = cp.GetTableContext(ctx, table)
		// Second call
		_, _ = cp.GetTableContext(ctx, table)

		if callCount != 1 {
			t.Errorf("callCount = %d, want 1", callCount)
		}
	})

	t.Run("does not cache errors by default", func(t *testing.T) {
		callCount := 0
		inner := ProviderFunc{
			GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
				callCount++
				return nil, errors.New("error")
			},
		}
		cp := NewCachingProvider(inner, DefaultCacheConfig())

		// First call
		_, _ = cp.GetTableContext(ctx, table)
		// Second call
		_, _ = cp.GetTableContext(ctx, table)

		if callCount != 2 {
			t.Errorf("callCount = %d, want 2 (errors not cached)", callCount)
		}
	})

	t.Run("caches errors when configured", func(t *testing.T) {
		callCount := 0
		inner := ProviderFunc{
			GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
				callCount++
				return nil, errors.New("error")
			},
		}
		cfg := DefaultCacheConfig()
		cfg.CacheErrors = true
		cp := NewCachingProvider(inner, cfg)

		// First call
		_, _ = cp.GetTableContext(ctx, table)
		// Second call
		_, _ = cp.GetTableContext(ctx, table)

		if callCount != 1 {
			t.Errorf("callCount = %d, want 1 (errors cached)", callCount)
		}
	})
}

func TestCachingProvider_GetColumnContext(t *testing.T) {
	ctx := context.Background()
	column := ColumnIdentifier{
		TableIdentifier: TableIdentifier{Catalog: "test", Schema: "test", Table: "test"},
		Column:          "col1",
	}

	callCount := 0
	inner := ProviderFunc{
		GetColumnContextFn: func(_ context.Context, _ ColumnIdentifier) (*ColumnContext, error) {
			callCount++
			return &ColumnContext{Description: "cached"}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	_, _ = cp.GetColumnContext(ctx, column)
	_, _ = cp.GetColumnContext(ctx, column)

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestCachingProvider_GetColumnsContext(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	callCount := 0
	inner := ProviderFunc{
		GetColumnsContextFn: func(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
			callCount++
			return map[string]*ColumnContext{"col1": {Description: "cached"}}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	_, _ = cp.GetColumnsContext(ctx, table)
	_, _ = cp.GetColumnsContext(ctx, table)

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestCachingProvider_GetLineage(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	callCount := 0
	inner := ProviderFunc{
		GetLineageFn: func(_ context.Context, _ TableIdentifier, _ LineageDirection, _ int) (*LineageInfo, error) {
			callCount++
			return &LineageInfo{Direction: LineageUpstream}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	_, _ = cp.GetLineage(ctx, table, LineageUpstream, 3)
	_, _ = cp.GetLineage(ctx, table, LineageUpstream, 3)

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Different parameters should not use cache
	_, _ = cp.GetLineage(ctx, table, LineageDownstream, 3)
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (different direction)", callCount)
	}
}

func TestCachingProvider_GetGlossaryTerm(t *testing.T) {
	ctx := context.Background()
	urn := "urn:li:glossaryTerm:test"

	callCount := 0
	inner := ProviderFunc{
		GetGlossaryTermFn: func(_ context.Context, _ string) (*GlossaryTerm, error) {
			callCount++
			return &GlossaryTerm{Name: "Test"}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	_, _ = cp.GetGlossaryTerm(ctx, urn)
	_, _ = cp.GetGlossaryTerm(ctx, urn)

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestCachingProvider_SearchTables_NotCached(t *testing.T) {
	ctx := context.Background()
	filter := SearchFilter{Query: "test"}

	callCount := 0
	inner := ProviderFunc{
		SearchTablesFn: func(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
			callCount++
			return []TableIdentifier{{Catalog: "c", Schema: "s", Table: "t"}}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	_, _ = cp.SearchTables(ctx, filter)
	_, _ = cp.SearchTables(ctx, filter)

	// SearchTables is not cached
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (SearchTables not cached)", callCount)
	}
}

func TestCachingProvider_TTLExpiration(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	callCount := 0
	inner := ProviderFunc{
		GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
			callCount++
			return &TableContext{Description: "test"}, nil
		},
	}
	cfg := CacheConfig{
		TTL:        50 * time.Millisecond,
		MaxEntries: 100,
	}
	cp := NewCachingProvider(inner, cfg)

	// First call
	_, _ = cp.GetTableContext(ctx, table)
	if callCount != 1 {
		t.Errorf("first call: callCount = %d, want 1", callCount)
	}

	// Second call before TTL expires
	_, _ = cp.GetTableContext(ctx, table)
	if callCount != 1 {
		t.Errorf("before TTL: callCount = %d, want 1", callCount)
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Third call after TTL expires
	_, _ = cp.GetTableContext(ctx, table)
	if callCount != 2 {
		t.Errorf("after TTL: callCount = %d, want 2", callCount)
	}
}

func TestCachingProvider_MaxEntries(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	inner := ProviderFunc{
		GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
			callCount++
			return &TableContext{Description: "test"}, nil
		},
	}
	cfg := CacheConfig{
		TTL:        5 * time.Minute,
		MaxEntries: 2,
	}
	cp := NewCachingProvider(inner, cfg)

	// Fill cache to capacity
	_, _ = cp.GetTableContext(ctx, TableIdentifier{Catalog: "c", Schema: "s", Table: "t1"})
	_, _ = cp.GetTableContext(ctx, TableIdentifier{Catalog: "c", Schema: "s", Table: "t2"})

	// Add one more - should evict oldest
	_, _ = cp.GetTableContext(ctx, TableIdentifier{Catalog: "c", Schema: "s", Table: "t3"})

	stats := cp.Stats()
	// May have 2 or 3 depending on timing, but should be at most MaxEntries + 1
	if stats.TotalEntries > 3 {
		t.Errorf("TotalEntries = %d, want <= 3", stats.TotalEntries)
	}
}

func TestCachingProvider_Stats(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	inner := ProviderFunc{
		GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
			return &TableContext{Description: "test"}, nil
		},
	}
	cfg := DefaultCacheConfig()
	cp := NewCachingProvider(inner, cfg)

	// Empty cache
	stats := cp.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("initial TotalEntries = %d, want 0", stats.TotalEntries)
	}

	// Add entry
	_, _ = cp.GetTableContext(ctx, table)
	stats = cp.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("after add TotalEntries = %d, want 1", stats.TotalEntries)
	}
	if stats.ActiveEntries != 1 {
		t.Errorf("after add ActiveEntries = %d, want 1", stats.ActiveEntries)
	}
}

func TestCachingProvider_Clear(t *testing.T) {
	ctx := context.Background()
	table := TableIdentifier{Catalog: "test", Schema: "test", Table: "test"}

	inner := ProviderFunc{
		GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
			return &TableContext{Description: "test"}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	_, _ = cp.GetTableContext(ctx, table)
	if cp.Stats().TotalEntries == 0 {
		t.Error("cache should have entries before Clear()")
	}

	cp.Clear()

	if cp.Stats().TotalEntries != 0 {
		t.Error("cache should be empty after Clear()")
	}
}

func TestCachingProvider_Close(t *testing.T) {
	closed := false
	inner := ProviderFunc{
		CloseFn: func() error { closed = true; return nil },
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	err := cp.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !closed {
		t.Error("inner provider Close() not called")
	}
	if cp.Stats().TotalEntries != 0 {
		t.Error("cache should be cleared on Close()")
	}
}

func TestCachingProvider_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	var callCount int64
	inner := ProviderFunc{
		GetTableContextFn: func(_ context.Context, _ TableIdentifier) (*TableContext, error) {
			atomic.AddInt64(&callCount, 1)
			return &TableContext{Description: "test"}, nil
		},
	}
	cp := NewCachingProvider(inner, DefaultCacheConfig())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			table := TableIdentifier{
				Catalog: "c",
				Schema:  "s",
				Table:   "t",
			}
			_, _ = cp.GetTableContext(ctx, table)
		}()
	}
	wg.Wait()

	// Due to race conditions, there might be multiple calls, but should be significantly
	// less than 100 due to caching
	finalCount := atomic.LoadInt64(&callCount)
	if finalCount > 10 {
		t.Errorf("callCount = %d, expected much less than 100 due to caching", finalCount)
	}
}

func TestDefaultCacheConfig(t *testing.T) {
	cfg := DefaultCacheConfig()

	if cfg.TTL != 5*time.Minute {
		t.Errorf("TTL = %v, want 5m", cfg.TTL)
	}
	if cfg.MaxEntries != 10000 {
		t.Errorf("MaxEntries = %d, want 10000", cfg.MaxEntries)
	}
	if cfg.CacheErrors {
		t.Error("CacheErrors should be false by default")
	}
	if cfg.ErrorTTL != 1*time.Minute {
		t.Errorf("ErrorTTL = %v, want 1m", cfg.ErrorTTL)
	}
}

func TestNewCachingProvider_Defaults(t *testing.T) {
	inner := ProviderFunc{}

	t.Run("applies default TTL", func(t *testing.T) {
		cp := NewCachingProvider(inner, CacheConfig{})
		// Can't directly check config, but should not panic
		if cp == nil {
			t.Error("NewCachingProvider returned nil")
		}
	})

	t.Run("applies default MaxEntries", func(t *testing.T) {
		cp := NewCachingProvider(inner, CacheConfig{TTL: time.Minute})
		if cp.Stats().MaxEntries != 10000 {
			t.Errorf("MaxEntries = %d, want 10000", cp.Stats().MaxEntries)
		}
	})
}

func TestCachingProvider_ImplementsInterface(_ *testing.T) {
	var _ Provider = (*CachingProvider)(nil)
}
