package semantic

import "context"

// ProviderFunc provides a function-based Provider implementation.
// Use this for simple providers that don't need struct state or only
// implement a subset of methods.
//
// Example:
//
//	provider := semantic.ProviderFunc{
//	    NameFn: func() string { return "inline" },
//	    GetTableContextFn: func(ctx context.Context, table TableIdentifier) (*TableContext, error) {
//	        return &TableContext{Description: "inline description"}, nil
//	    },
//	}
//
//nolint:dupl // mirrors Provider interface by design
type ProviderFunc struct {
	// NameFn returns the provider name. Required.
	NameFn func() string

	// GetTableContextFn retrieves table metadata. Optional.
	GetTableContextFn func(ctx context.Context, table TableIdentifier) (*TableContext, error)

	// GetColumnContextFn retrieves column metadata. Optional.
	GetColumnContextFn func(ctx context.Context, column ColumnIdentifier) (*ColumnContext, error)

	// GetColumnsContextFn retrieves all column metadata for a table. Optional.
	GetColumnsContextFn func(ctx context.Context, table TableIdentifier) (map[string]*ColumnContext, error)

	// GetLineageFn retrieves lineage. Optional.
	GetLineageFn func(ctx context.Context, table TableIdentifier, direction LineageDirection, maxDepth int) (*LineageInfo, error)

	// GetGlossaryTermFn retrieves a glossary term. Optional.
	GetGlossaryTermFn func(ctx context.Context, urn string) (*GlossaryTerm, error)

	// SearchTablesFn searches tables. Optional.
	SearchTablesFn func(ctx context.Context, filter SearchFilter) ([]TableIdentifier, error)

	// CloseFn releases resources. Optional.
	CloseFn func() error
}

// Name implements Provider.
func (pf ProviderFunc) Name() string {
	if pf.NameFn != nil {
		return pf.NameFn()
	}
	return "func"
}

// GetTableContext implements Provider.
func (pf ProviderFunc) GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error) {
	if pf.GetTableContextFn != nil {
		return pf.GetTableContextFn(ctx, table)
	}
	return nil, nil
}

// GetColumnContext implements Provider.
func (pf ProviderFunc) GetColumnContext(ctx context.Context, column ColumnIdentifier) (*ColumnContext, error) {
	if pf.GetColumnContextFn != nil {
		return pf.GetColumnContextFn(ctx, column)
	}
	return nil, nil
}

// GetColumnsContext implements Provider.
func (pf ProviderFunc) GetColumnsContext(ctx context.Context, table TableIdentifier) (map[string]*ColumnContext, error) {
	if pf.GetColumnsContextFn != nil {
		return pf.GetColumnsContextFn(ctx, table)
	}
	return nil, nil
}

// GetLineage implements Provider.
func (pf ProviderFunc) GetLineage(
	ctx context.Context, table TableIdentifier, direction LineageDirection, maxDepth int,
) (*LineageInfo, error) {
	if pf.GetLineageFn != nil {
		return pf.GetLineageFn(ctx, table, direction, maxDepth)
	}
	return nil, nil
}

// GetGlossaryTerm implements Provider.
func (pf ProviderFunc) GetGlossaryTerm(ctx context.Context, urn string) (*GlossaryTerm, error) {
	if pf.GetGlossaryTermFn != nil {
		return pf.GetGlossaryTermFn(ctx, urn)
	}
	return nil, nil
}

// SearchTables implements Provider.
func (pf ProviderFunc) SearchTables(ctx context.Context, filter SearchFilter) ([]TableIdentifier, error) {
	if pf.SearchTablesFn != nil {
		return pf.SearchTablesFn(ctx, filter)
	}
	return nil, nil
}

// Close implements Provider.
func (pf ProviderFunc) Close() error {
	if pf.CloseFn != nil {
		return pf.CloseFn()
	}
	return nil
}

// Verify ProviderFunc implements Provider.
var _ Provider = ProviderFunc{}
