// Package static provides a file-based Provider implementation.
// It loads semantic metadata from YAML or JSON files, making it ideal
// for testing, development, and simple deployments.
package static

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/txn2/mcp-trino/pkg/semantic"
)

// ProviderName is the name returned by Provider.Name().
const ProviderName = "static"

// Common errors.
var (
	ErrNoFilePath        = errors.New("static: file path is required")
	ErrFileNotFound      = errors.New("static: file not found")
	ErrUnsupportedFormat = errors.New("static: unsupported file format (use .yaml, .yml, or .json)")
)

// Provider loads semantic metadata from YAML or JSON files.
type Provider struct {
	config Config

	// Indexed data for fast lookups
	tables   map[string]*TableEntry    // key: table identifier string
	columns  map[string]*ColumnEntry   // key: column identifier string
	glossary map[string]*GlossaryEntry // key: URN

	mu       sync.RWMutex
	stopChan chan struct{}
	stopped  bool
}

// New creates a new static file provider.
func New(cfg Config) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Provider{
		config:   cfg,
		tables:   make(map[string]*TableEntry),
		columns:  make(map[string]*ColumnEntry),
		glossary: make(map[string]*GlossaryEntry),
		stopChan: make(chan struct{}),
	}

	// Load initial data
	if err := p.load(); err != nil {
		return nil, err
	}

	// Start watcher if configured
	if cfg.WatchInterval > 0 {
		go p.watch()
	}

	return p, nil
}

// Name implements semantic.Provider.
func (p *Provider) Name() string {
	return ProviderName
}

// GetTableContext implements semantic.Provider.
func (p *Provider) GetTableContext(_ context.Context, table semantic.TableIdentifier) (*semantic.TableContext, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entry, ok := p.tables[table.Key()]
	if !ok {
		return nil, nil
	}

	ctx := entry.toTableContext()
	ctx.FetchedAt = time.Now()
	return ctx, nil
}

// GetColumnContext implements semantic.Provider.
func (p *Provider) GetColumnContext(_ context.Context, column semantic.ColumnIdentifier) (*semantic.ColumnContext, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// First find the table (use TableIdentifier.Key() explicitly for clarity)
	//nolint:staticcheck // SA1019: explicit access to embedded field for clarity
	tableEntry, ok := p.tables[column.TableIdentifier.Key()]
	if !ok {
		return nil, nil
	}

	// Then find the column within the table
	colEntry, ok := tableEntry.Columns[column.Column]
	if !ok {
		return nil, nil
	}

	return colEntry.toColumnContext(column.TableIdentifier, column.Column), nil
}

// GetColumnsContext implements semantic.Provider.
func (p *Provider) GetColumnsContext(_ context.Context, table semantic.TableIdentifier) (map[string]*semantic.ColumnContext, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tableEntry, ok := p.tables[table.Key()]
	if !ok {
		return nil, nil
	}

	if len(tableEntry.Columns) == 0 {
		return nil, nil
	}

	result := make(map[string]*semantic.ColumnContext, len(tableEntry.Columns))
	for name, entry := range tableEntry.Columns {
		result[name] = entry.toColumnContext(table, name)
	}

	return result, nil
}

// GetLineage implements semantic.Provider.
// Static provider does not support lineage; returns nil.
func (p *Provider) GetLineage(
	_ context.Context, _ semantic.TableIdentifier,
	_ semantic.LineageDirection, _ int,
) (*semantic.LineageInfo, error) {
	return nil, nil
}

// GetGlossaryTerm implements semantic.Provider.
func (p *Provider) GetGlossaryTerm(_ context.Context, urn string) (*semantic.GlossaryTerm, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entry, ok := p.glossary[urn]
	if !ok {
		return nil, nil
	}

	return entry.toGlossaryTerm(), nil
}

// SearchTables implements semantic.Provider.
func (p *Provider) SearchTables(_ context.Context, filter semantic.SearchFilter) ([]semantic.TableIdentifier, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []semantic.TableIdentifier

	for _, entry := range p.tables {
		if p.matchesFilter(entry, filter) {
			results = append(results, semantic.TableIdentifier{
				Connection: entry.Connection,
				Catalog:    entry.Catalog,
				Schema:     entry.Schema,
				Table:      entry.Table,
			})
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

// Close implements semantic.Provider.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped {
		p.stopped = true
		close(p.stopChan)
	}
	return nil
}

// Reload forces a reload of the metadata file.
func (p *Provider) Reload() error {
	return p.load()
}

// load reads and parses the metadata file.
func (p *Provider) load() error {
	data, err := os.ReadFile(p.config.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}

	var file File
	ext := strings.ToLower(filepath.Ext(p.config.FilePath))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &file); err != nil {
			return err
		}
	case ".json":
		if err := json.Unmarshal(data, &file); err != nil {
			return err
		}
	default:
		return ErrUnsupportedFormat
	}

	// Build indexes
	tables := make(map[string]*TableEntry, len(file.Tables))
	for i := range file.Tables {
		entry := &file.Tables[i]
		id := semantic.TableIdentifier{
			Connection: entry.Connection,
			Catalog:    entry.Catalog,
			Schema:     entry.Schema,
			Table:      entry.Table,
		}
		tables[id.Key()] = entry
	}

	glossary := make(map[string]*GlossaryEntry, len(file.Glossary))
	for i := range file.Glossary {
		entry := &file.Glossary[i]
		glossary[entry.URN] = entry
	}

	// Atomically update
	p.mu.Lock()
	p.tables = tables
	p.glossary = glossary
	p.mu.Unlock()

	return nil
}

// watch periodically reloads the file.
func (p *Provider) watch() {
	ticker := time.NewTicker(p.config.WatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = p.load() //nolint:errcheck // reload errors are non-fatal, continue watching
		case <-p.stopChan:
			return
		}
	}
}

// matchesFilter checks if a table entry matches the search filter.
func (p *Provider) matchesFilter(entry *TableEntry, filter semantic.SearchFilter) bool {
	return matchesDeprecation(entry, filter) &&
		matchesCatalogSchema(entry, filter) &&
		matchesDomain(entry, filter) &&
		matchesOwner(entry, filter) &&
		matchesTags(entry, filter) &&
		matchesQuery(entry, filter)
}

// matchesDeprecation checks the deprecation filter.
func matchesDeprecation(entry *TableEntry, filter semantic.SearchFilter) bool {
	return filter.IncludeDeprecated || !entry.Deprecated
}

// matchesCatalogSchema checks catalog and schema filters.
func matchesCatalogSchema(entry *TableEntry, filter semantic.SearchFilter) bool {
	if filter.Catalog != "" && !strings.EqualFold(entry.Catalog, filter.Catalog) {
		return false
	}
	if filter.Schema != "" && !strings.EqualFold(entry.Schema, filter.Schema) {
		return false
	}
	return true
}

// matchesDomain checks the domain filter.
func matchesDomain(entry *TableEntry, filter semantic.SearchFilter) bool {
	if filter.Domain == "" {
		return true
	}
	if entry.Domain == nil {
		return false
	}
	return strings.EqualFold(entry.Domain.Name, filter.Domain) || entry.Domain.URN == filter.Domain
}

// matchesOwner checks the owner filter.
func matchesOwner(entry *TableEntry, filter semantic.SearchFilter) bool {
	if filter.Owner == "" {
		return true
	}
	for _, owner := range entry.Owners {
		if strings.EqualFold(owner.ID, filter.Owner) || strings.EqualFold(owner.Name, filter.Owner) {
			return true
		}
	}
	return false
}

// matchesTags checks the tag filter.
func matchesTags(entry *TableEntry, filter semantic.SearchFilter) bool {
	if len(filter.Tags) == 0 {
		return true
	}
	entryTags := make(map[string]bool, len(entry.Tags))
	for _, t := range entry.Tags {
		entryTags[strings.ToLower(t.Name)] = true
	}
	for _, required := range filter.Tags {
		if !entryTags[strings.ToLower(required)] {
			return false
		}
	}
	return true
}

// matchesQuery checks the query filter (search in name and description).
func matchesQuery(entry *TableEntry, filter semantic.SearchFilter) bool {
	if filter.Query == "" {
		return true
	}
	query := strings.ToLower(filter.Query)
	return strings.Contains(strings.ToLower(entry.Table), query) ||
		strings.Contains(strings.ToLower(entry.Description), query)
}

// TableCount returns the number of tables loaded.
func (p *Provider) TableCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.tables)
}

// GlossaryCount returns the number of glossary terms loaded.
func (p *Provider) GlossaryCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.glossary)
}

// Verify Provider implements semantic.Provider.
var _ semantic.Provider = (*Provider)(nil)
