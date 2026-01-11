package semantic

import "context"

// NoOpProvider is a Provider that returns no metadata.
// It is used as the default when no semantic layer is configured,
// ensuring zero overhead (no allocations, no external calls).
type NoOpProvider struct{}

// Name implements Provider.
func (n *NoOpProvider) Name() string {
	return "noop"
}

// GetTableContext implements Provider.
func (n *NoOpProvider) GetTableContext(_ context.Context, _ TableIdentifier) (*TableContext, error) {
	return nil, nil
}

// GetColumnContext implements Provider.
func (n *NoOpProvider) GetColumnContext(_ context.Context, _ ColumnIdentifier) (*ColumnContext, error) {
	return nil, nil
}

// GetColumnsContext implements Provider.
func (n *NoOpProvider) GetColumnsContext(_ context.Context, _ TableIdentifier) (map[string]*ColumnContext, error) {
	return nil, nil
}

// GetLineage implements Provider.
func (n *NoOpProvider) GetLineage(_ context.Context, _ TableIdentifier, _ LineageDirection, _ int) (*LineageInfo, error) {
	return nil, nil
}

// GetGlossaryTerm implements Provider.
func (n *NoOpProvider) GetGlossaryTerm(_ context.Context, _ string) (*GlossaryTerm, error) {
	return nil, nil
}

// SearchTables implements Provider.
func (n *NoOpProvider) SearchTables(_ context.Context, _ SearchFilter) ([]TableIdentifier, error) {
	return nil, nil
}

// Close implements Provider.
func (n *NoOpProvider) Close() error {
	return nil
}

// Verify NoOpProvider implements Provider.
var _ Provider = (*NoOpProvider)(nil)
