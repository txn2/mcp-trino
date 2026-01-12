package semantic

import (
	"context"
	"strings"
)

// ProviderChain chains multiple Providers together.
// It queries providers in order and returns the first non-nil result.
// This enables composing multiple sources (e.g., custom overrides + DataHub).
//
// Example:
//
//	chain := semantic.NewProviderChain(
//	    customOverrides,  // Check custom overrides first
//	    datahubProvider,  // Fall back to DataHub
//	)
type ProviderChain struct {
	providers []Provider
}

// NewProviderChain creates a provider chain from the given providers.
// Providers are queried in order; the first non-nil result wins.
func NewProviderChain(providers ...Provider) *ProviderChain {
	return &ProviderChain{providers: providers}
}

// Name returns a composite name of all chained providers.
func (pc *ProviderChain) Name() string {
	if len(pc.providers) == 0 {
		return "chain(empty)"
	}
	if len(pc.providers) == 1 {
		return pc.providers[0].Name()
	}

	names := make([]string, len(pc.providers))
	for i, p := range pc.providers {
		names[i] = p.Name()
	}
	return "chain(" + strings.Join(names, ",") + ")"
}

// GetTableContext queries providers in order, returning the first non-nil result.
func (pc *ProviderChain) GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error) {
	for _, p := range pc.providers {
		result, err := p.GetTableContext(ctx, table)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

// GetColumnContext queries providers in order, returning the first non-nil result.
func (pc *ProviderChain) GetColumnContext(ctx context.Context, column ColumnIdentifier) (*ColumnContext, error) {
	for _, p := range pc.providers {
		result, err := p.GetColumnContext(ctx, column)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

// GetColumnsContext merges results from all providers.
// Later providers can override earlier ones for the same column.
func (pc *ProviderChain) GetColumnsContext(ctx context.Context, table TableIdentifier) (map[string]*ColumnContext, error) {
	merged := make(map[string]*ColumnContext)

	for _, p := range pc.providers {
		result, err := p.GetColumnsContext(ctx, table)
		if err != nil {
			return nil, err
		}
		for name, col := range result {
			merged[name] = col
		}
	}

	if len(merged) == 0 {
		return nil, nil
	}
	return merged, nil
}

// GetLineage queries providers in order, returning the first non-nil result.
func (pc *ProviderChain) GetLineage(
	ctx context.Context, table TableIdentifier, direction LineageDirection, maxDepth int,
) (*LineageInfo, error) {
	for _, p := range pc.providers {
		result, err := p.GetLineage(ctx, table, direction, maxDepth)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

// GetGlossaryTerm queries providers in order, returning the first non-nil result.
func (pc *ProviderChain) GetGlossaryTerm(ctx context.Context, urn string) (*GlossaryTerm, error) {
	for _, p := range pc.providers {
		result, err := p.GetGlossaryTerm(ctx, urn)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

// SearchTables combines results from all providers (union).
// Duplicate tables are deduplicated.
func (pc *ProviderChain) SearchTables(ctx context.Context, filter SearchFilter) ([]TableIdentifier, error) {
	seen := make(map[string]bool)
	var results []TableIdentifier

	for _, p := range pc.providers {
		tables, err := p.SearchTables(ctx, filter)
		if err != nil {
			return nil, err
		}
		for _, t := range tables {
			key := t.Key()
			if !seen[key] {
				seen[key] = true
				results = append(results, t)
			}
		}
	}

	return results, nil
}

// Close closes all chained providers.
func (pc *ProviderChain) Close() error {
	var firstErr error
	for _, p := range pc.providers {
		if err := p.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Append adds providers to the chain.
func (pc *ProviderChain) Append(providers ...Provider) {
	pc.providers = append(pc.providers, providers...)
}

// Len returns the number of providers in the chain.
func (pc *ProviderChain) Len() int {
	return len(pc.providers)
}

// Verify ProviderChain implements Provider.
var _ Provider = (*ProviderChain)(nil)
