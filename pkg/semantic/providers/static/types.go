package static

import (
	"github.com/txn2/mcp-trino/pkg/semantic"
)

// File represents the structure of a semantic metadata file.
type File struct {
	// Tables contains table-level metadata.
	Tables []TableEntry `json:"tables" yaml:"tables"`

	// Glossary contains glossary term definitions.
	Glossary []GlossaryEntry `json:"glossary" yaml:"glossary"`
}

// TableEntry represents a single table's metadata in the file.
type TableEntry struct {
	// Connection is the Trino connection name (optional).
	Connection string `json:"connection,omitempty" yaml:"connection,omitempty"`

	// Catalog is the catalog name.
	Catalog string `json:"catalog" yaml:"catalog"`

	// Schema is the schema name.
	Schema string `json:"schema" yaml:"schema"`

	// Table is the table name.
	Table string `json:"table" yaml:"table"`

	// Description is the business description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Owners contains ownership information.
	Owners []OwnerEntry `json:"owners,omitempty" yaml:"owners,omitempty"`

	// Tags are labels attached to the table.
	Tags []TagEntry `json:"tags,omitempty" yaml:"tags,omitempty"`

	// GlossaryTerms are URNs of associated glossary terms.
	GlossaryTerms []string `json:"glossary_terms,omitempty" yaml:"glossary_terms,omitempty"`

	// Domain is the data domain.
	Domain *DomainEntry `json:"domain,omitempty" yaml:"domain,omitempty"`

	// IsDeprecated indicates if the table is deprecated.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// DeprecationNote explains why the table is deprecated.
	DeprecationNote string `json:"deprecation_note,omitempty" yaml:"deprecation_note,omitempty"`

	// ReplacedBy indicates what table replaces this one.
	ReplacedBy string `json:"replaced_by,omitempty" yaml:"replaced_by,omitempty"`

	// Columns contains column-level metadata.
	Columns map[string]ColumnEntry `json:"columns,omitempty" yaml:"columns,omitempty"`

	// CustomProperties holds additional metadata.
	CustomProperties map[string]any `json:"custom_properties,omitempty" yaml:"custom_properties,omitempty"`
}

// OwnerEntry represents an owner in the file format.
type OwnerEntry struct {
	ID   string `json:"id,omitempty" yaml:"id,omitempty"`
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"` // "user" or "group"
	Role string `json:"role,omitempty" yaml:"role,omitempty"`
}

// TagEntry represents a tag in the file format.
type TagEntry struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// DomainEntry represents a domain in the file format.
type DomainEntry struct {
	URN         string `json:"urn,omitempty" yaml:"urn,omitempty"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ColumnEntry represents column metadata in the file format.
type ColumnEntry struct {
	Description      string   `json:"description,omitempty" yaml:"description,omitempty"`
	Tags             []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	GlossaryTerms    []string `json:"glossary_terms,omitempty" yaml:"glossary_terms,omitempty"`
	Sensitive        bool     `json:"sensitive,omitempty" yaml:"sensitive,omitempty"`
	SensitivityLevel string   `json:"sensitivity_level,omitempty" yaml:"sensitivity_level,omitempty"`
}

// GlossaryEntry represents a glossary term in the file format.
type GlossaryEntry struct {
	URN          string   `json:"urn" yaml:"urn"`
	Name         string   `json:"name" yaml:"name"`
	Definition   string   `json:"definition,omitempty" yaml:"definition,omitempty"`
	RelatedTerms []string `json:"related_terms,omitempty" yaml:"related_terms,omitempty"`
}

// toTableContext converts a TableEntry to semantic.TableContext.
func (t *TableEntry) toTableContext() *semantic.TableContext {
	ctx := &semantic.TableContext{
		Identifier: semantic.TableIdentifier{
			Connection: t.Connection,
			Catalog:    t.Catalog,
			Schema:     t.Schema,
			Table:      t.Table,
		},
		Description:      t.Description,
		CustomProperties: t.CustomProperties,
		Source:           "static",
	}

	// Convert owners
	if len(t.Owners) > 0 {
		owners := make([]semantic.Owner, len(t.Owners))
		for i, o := range t.Owners {
			owners[i] = semantic.Owner{
				ID:   o.ID,
				Name: o.Name,
				Type: o.Type,
				Role: o.Role,
			}
		}
		ctx.Ownership = &semantic.Ownership{Owners: owners}
	}

	// Convert tags
	if len(t.Tags) > 0 {
		ctx.Tags = make([]semantic.Tag, len(t.Tags))
		for i, tag := range t.Tags {
			ctx.Tags[i] = semantic.Tag{
				Name:        tag.Name,
				Description: tag.Description,
				Source:      "static",
			}
		}
	}

	// Convert glossary terms (as references)
	if len(t.GlossaryTerms) > 0 {
		ctx.GlossaryTerms = make([]semantic.GlossaryTerm, len(t.GlossaryTerms))
		for i, urn := range t.GlossaryTerms {
			ctx.GlossaryTerms[i] = semantic.GlossaryTerm{
				URN:    urn,
				Source: "static",
			}
		}
	}

	// Convert domain
	if t.Domain != nil {
		ctx.Domain = &semantic.Domain{
			URN:         t.Domain.URN,
			Name:        t.Domain.Name,
			Description: t.Domain.Description,
		}
	}

	// Convert deprecation
	if t.Deprecated {
		ctx.Deprecation = &semantic.Deprecation{
			Deprecated: true,
			Note:       t.DeprecationNote,
			ReplacedBy: t.ReplacedBy,
		}
	}

	return ctx
}

// toColumnContext converts a ColumnEntry to semantic.ColumnContext.
func (c *ColumnEntry) toColumnContext(table semantic.TableIdentifier, columnName string) *semantic.ColumnContext {
	ctx := &semantic.ColumnContext{
		Identifier: semantic.ColumnIdentifier{
			TableIdentifier: table,
			Column:          columnName,
		},
		Description:      c.Description,
		IsSensitive:      c.Sensitive,
		SensitivityLevel: c.SensitivityLevel,
		Source:           "static",
	}

	// Convert tags
	if len(c.Tags) > 0 {
		ctx.Tags = make([]semantic.Tag, len(c.Tags))
		for i, tagName := range c.Tags {
			ctx.Tags[i] = semantic.Tag{
				Name:   tagName,
				Source: "static",
			}
		}
	}

	// Convert glossary terms (as references)
	if len(c.GlossaryTerms) > 0 {
		ctx.GlossaryTerms = make([]semantic.GlossaryTerm, len(c.GlossaryTerms))
		for i, urn := range c.GlossaryTerms {
			ctx.GlossaryTerms[i] = semantic.GlossaryTerm{
				URN:    urn,
				Source: "static",
			}
		}
	}

	return ctx
}

// toGlossaryTerm converts a GlossaryEntry to semantic.GlossaryTerm.
func (g *GlossaryEntry) toGlossaryTerm() *semantic.GlossaryTerm {
	return &semantic.GlossaryTerm{
		URN:          g.URN,
		Name:         g.Name,
		Definition:   g.Definition,
		RelatedTerms: g.RelatedTerms,
		Source:       "static",
	}
}
