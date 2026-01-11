package semantic

import (
	"context"
)

// Provider retrieves semantic metadata from external catalogs.
// Implementations may connect to DataHub, Atlan, OpenMetadata, Alation,
// dbt docs, or custom metadata sources.
//
// All methods should return nil (not an error) if metadata is not found.
// Errors should only be returned for connection/authentication failures
// or other exceptional conditions.
//
// Implementations must be safe for concurrent use.
//
//nolint:dupl // ProviderFunc mirrors this interface by design
type Provider interface {
	// Name returns the provider name (e.g., "datahub", "atlan", "static").
	// Used for logging, metrics, and source attribution.
	Name() string

	// GetTableContext retrieves semantic metadata for a table.
	// Returns nil if no metadata is found (not an error).
	GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error)

	// GetColumnContext retrieves semantic metadata for a column.
	// Returns nil if no metadata is found (not an error).
	GetColumnContext(ctx context.Context, column ColumnIdentifier) (*ColumnContext, error)

	// GetColumnsContext retrieves semantic metadata for all columns in a table.
	// Returns a map from column name to ColumnContext.
	// Returns an empty map if no metadata is found (not an error).
	GetColumnsContext(ctx context.Context, table TableIdentifier) (map[string]*ColumnContext, error)

	// GetLineage retrieves lineage information for a table.
	// The maxDepth parameter limits how many hops to traverse (0 = unlimited).
	// Returns nil if no lineage is found (not an error).
	GetLineage(ctx context.Context, table TableIdentifier, direction LineageDirection, maxDepth int) (*LineageInfo, error)

	// GetGlossaryTerm retrieves a glossary term by URN.
	// Returns nil if the term is not found (not an error).
	GetGlossaryTerm(ctx context.Context, urn string) (*GlossaryTerm, error)

	// SearchTables searches for tables matching the given filter.
	// Returns an empty slice if no matches are found (not an error).
	SearchTables(ctx context.Context, filter SearchFilter) ([]TableIdentifier, error)

	// Close releases any resources held by the provider.
	// Implementations should be idempotent.
	Close() error
}
