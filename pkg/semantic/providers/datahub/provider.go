// Package datahub provides a DataHub Provider implementation.
// It connects to DataHub's GraphQL API to retrieve semantic metadata
// including descriptions, ownership, tags, glossary terms, and lineage.
package datahub

import (
	"context"
	"fmt"

	"github.com/txn2/mcp-trino/pkg/semantic"
)

// ProviderName is the name returned by Provider.Name().
const ProviderName = "datahub"

// Provider implements semantic.Provider for DataHub.
type Provider struct {
	client   *Client
	platform string
	env      string
}

// New creates a new DataHub provider.
func New(cfg Config) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg = cfg.WithDefaults()

	return &Provider{
		client:   NewClient(cfg),
		platform: cfg.Platform,
		env:      cfg.Environment,
	}, nil
}

// Name implements semantic.Provider.
func (p *Provider) Name() string {
	return ProviderName
}

// GetTableContext implements semantic.Provider.
func (p *Provider) GetTableContext(ctx context.Context, table semantic.TableIdentifier) (*semantic.TableContext, error) {
	urn := buildDatasetURN(table, p.platform, p.env)

	var resp datasetResponse
	if err := p.client.Execute(ctx, getDatasetQuery, map[string]any{"urn": urn}, &resp); err != nil {
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}

	if resp.Dataset == nil {
		return nil, nil
	}

	return mapDatasetToTableContext(resp.Dataset, table), nil
}

// GetColumnContext implements semantic.Provider.
func (p *Provider) GetColumnContext(ctx context.Context, column semantic.ColumnIdentifier) (*semantic.ColumnContext, error) {
	columns, err := p.GetColumnsContext(ctx, column.TableIdentifier)
	if err != nil {
		return nil, err
	}
	if columns == nil {
		return nil, nil
	}
	return columns[column.Column], nil
}

// GetColumnsContext implements semantic.Provider.
func (p *Provider) GetColumnsContext(ctx context.Context, table semantic.TableIdentifier) (map[string]*semantic.ColumnContext, error) {
	urn := buildDatasetURN(table, p.platform, p.env)

	var resp schemaResponse
	if err := p.client.Execute(ctx, getSchemaQuery, map[string]any{"urn": urn}, &resp); err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if resp.Dataset == nil || resp.Dataset.SchemaMetadata == nil {
		return nil, nil
	}

	return mapSchemaToColumnContexts(resp.Dataset.SchemaMetadata, table), nil
}

// GetLineage implements semantic.Provider.
func (p *Provider) GetLineage(
	ctx context.Context, table semantic.TableIdentifier,
	direction semantic.LineageDirection, maxDepth int,
) (*semantic.LineageInfo, error) {
	urn := buildDatasetURN(table, p.platform, p.env)

	// Map direction to DataHub enum
	dhDirection := "UPSTREAM"
	if direction == semantic.LineageDownstream {
		dhDirection = "DOWNSTREAM"
	}

	var resp struct {
		SearchAcrossLineage struct {
			SearchResults []struct {
				Degree int `json:"degree"`
				Entity struct {
					URN      string `json:"urn"`
					Type     string `json:"type"`
					Name     string `json:"name"`
					Platform *struct {
						Name string `json:"name"`
					} `json:"platform"`
				} `json:"entity"`
			} `json:"searchResults"`
		} `json:"searchAcrossLineage"`
	}

	variables := map[string]any{
		"urn":       urn,
		"direction": dhDirection,
	}
	if maxDepth > 0 {
		variables["depth"] = maxDepth
	}

	if err := p.client.Execute(ctx, getLineageQuery, variables, &resp); err != nil {
		return nil, fmt.Errorf("failed to get lineage: %w", err)
	}

	if len(resp.SearchAcrossLineage.SearchResults) == 0 {
		return nil, nil
	}

	lineage := &semantic.LineageInfo{
		Table:     table,
		Direction: direction,
		Source:    ProviderName,
	}

	for _, result := range resp.SearchAcrossLineage.SearchResults {
		if result.Entity.Type != "DATASET" {
			continue
		}

		targetTable, err := parseDatasetURN(result.Entity.URN)
		if err != nil {
			continue // Skip unparseable URNs
		}

		edge := semantic.LineageEdge{}
		if direction == semantic.LineageUpstream {
			edge.SourceTable = *targetTable
			edge.TargetTable = table
		} else {
			edge.SourceTable = table
			edge.TargetTable = *targetTable
		}

		lineage.Edges = append(lineage.Edges, edge)
	}

	return lineage, nil
}

// GetGlossaryTerm implements semantic.Provider.
func (p *Provider) GetGlossaryTerm(ctx context.Context, urn string) (*semantic.GlossaryTerm, error) {
	var resp glossaryTermResponse
	if err := p.client.Execute(ctx, getGlossaryTermQuery, map[string]any{"urn": urn}, &resp); err != nil {
		return nil, fmt.Errorf("failed to get glossary term: %w", err)
	}

	if resp.GlossaryTerm == nil {
		return nil, nil
	}

	return mapGlossaryTermData(resp.GlossaryTerm), nil
}

// SearchTables implements semantic.Provider.
func (p *Provider) SearchTables(ctx context.Context, filter semantic.SearchFilter) ([]semantic.TableIdentifier, error) {
	query := "*"
	if filter.Query != "" {
		query = filter.Query
	}

	count := 100
	if filter.Limit > 0 && filter.Limit < count {
		count = filter.Limit
	}

	// Build filters
	var filters []map[string]any
	if filter.Domain != "" {
		filters = append(filters, map[string]any{
			"field":  "domains",
			"values": []string{filter.Domain},
		})
	}
	if len(filter.Tags) > 0 {
		filters = append(filters, map[string]any{
			"field":  "tags",
			"values": filter.Tags,
		})
	}

	variables := map[string]any{
		"query": query,
		"count": count,
	}
	if len(filters) > 0 {
		variables["filters"] = filters
	}

	var resp searchResponse
	if err := p.client.Execute(ctx, searchDatasetsQuery, variables, &resp); err != nil {
		return nil, fmt.Errorf("failed to search datasets: %w", err)
	}

	results := make([]semantic.TableIdentifier, 0, len(resp.Search.SearchResults))
	for _, result := range resp.Search.SearchResults {
		table, err := parseDatasetURN(result.Entity.URN)
		if err != nil {
			continue // Skip unparseable URNs
		}
		results = append(results, *table)
	}

	return results, nil
}

// Close implements semantic.Provider.
func (p *Provider) Close() error {
	return nil
}

// Verify Provider implements semantic.Provider.
var _ semantic.Provider = (*Provider)(nil)
