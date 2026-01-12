package semantic

import (
	"fmt"
	"time"
)

// TableIdentifier uniquely identifies a table across Trino connections.
type TableIdentifier struct {
	// Connection is the Trino connection name (empty for default).
	Connection string `json:"connection,omitempty" yaml:"connection,omitempty"`

	// Catalog is the Trino catalog name.
	Catalog string `json:"catalog" yaml:"catalog"`

	// Schema is the Trino schema name.
	Schema string `json:"schema" yaml:"schema"`

	// Table is the table name.
	Table string `json:"table" yaml:"table"`
}

// String returns the fully-qualified table name.
func (t TableIdentifier) String() string {
	if t.Connection != "" {
		return fmt.Sprintf("%s:%s.%s.%s", t.Connection, t.Catalog, t.Schema, t.Table)
	}
	return fmt.Sprintf("%s.%s.%s", t.Catalog, t.Schema, t.Table)
}

// Key returns a unique key for map lookups.
func (t TableIdentifier) Key() string {
	return t.String()
}

// ColumnIdentifier uniquely identifies a column.
type ColumnIdentifier struct {
	TableIdentifier `json:",inline" yaml:",inline"`

	// Column is the column name.
	Column string `json:"column" yaml:"column"`
}

// String returns the fully-qualified column name.
func (c ColumnIdentifier) String() string {
	return fmt.Sprintf("%s.%s", c.TableIdentifier.String(), c.Column)
}

// Ownership represents ownership information for a data asset.
type Ownership struct {
	// Owners is the list of owners with their roles.
	Owners []Owner `json:"owners,omitempty" yaml:"owners,omitempty"`
}

// Owner represents a single owner of a data asset.
type Owner struct {
	// ID is the unique identifier (e.g., email, LDAP DN).
	ID string `json:"id,omitempty" yaml:"id,omitempty"`

	// Name is the display name.
	Name string `json:"name" yaml:"name"`

	// Type is "user" or "group".
	Type string `json:"type" yaml:"type"`

	// Role describes the ownership role (e.g., "Data Steward", "Technical Owner").
	Role string `json:"role,omitempty" yaml:"role,omitempty"`
}

// Tag represents a tag or label attached to a data asset.
type Tag struct {
	// Name is the tag name.
	Name string `json:"name" yaml:"name"`

	// Description provides context for the tag.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Source indicates where the tag came from (e.g., "datahub", "manual").
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// GlossaryTerm represents a business glossary term.
type GlossaryTerm struct {
	// URN is the unique identifier for the term.
	URN string `json:"urn" yaml:"urn"`

	// Name is the term name.
	Name string `json:"name" yaml:"name"`

	// Definition is the business definition.
	Definition string `json:"definition,omitempty" yaml:"definition,omitempty"`

	// RelatedTerms are URNs of related terms.
	RelatedTerms []string `json:"related_terms,omitempty" yaml:"related_terms,omitempty"`

	// Source indicates the glossary source.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// Domain represents a data domain classification.
type Domain struct {
	// URN is the unique identifier.
	URN string `json:"urn,omitempty" yaml:"urn,omitempty"`

	// Name is the domain name.
	Name string `json:"name" yaml:"name"`

	// Description provides context for the domain.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Deprecation indicates if a data asset is deprecated.
type Deprecation struct {
	// Deprecated is true if the asset is deprecated.
	Deprecated bool `json:"deprecated" yaml:"deprecated"`

	// Note explains why the asset is deprecated.
	Note string `json:"note,omitempty" yaml:"note,omitempty"`

	// ReplacedBy is the identifier of the replacement asset.
	ReplacedBy string `json:"replaced_by,omitempty" yaml:"replaced_by,omitempty"`

	// DecommissionTime is when the asset will be removed.
	DecommissionTime *time.Time `json:"decommission_time,omitempty" yaml:"decommission_time,omitempty"`
}

// DataQuality contains data quality metrics for an asset.
type DataQuality struct {
	// Score is the overall quality score (0-100).
	Score *float64 `json:"score,omitempty" yaml:"score,omitempty"`

	// LastAssessed is when quality was last assessed.
	LastAssessed *time.Time `json:"last_assessed,omitempty" yaml:"last_assessed,omitempty"`

	// Freshness indicates how current the data is.
	Freshness *FreshnessInfo `json:"freshness,omitempty" yaml:"freshness,omitempty"`

	// Rules are the quality rules applied.
	Rules []QualityRule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// FreshnessInfo describes data freshness.
type FreshnessInfo struct {
	// LastUpdated is when the data was last updated.
	LastUpdated *time.Time `json:"last_updated,omitempty" yaml:"last_updated,omitempty"`

	// ExpectedFrequency is how often updates are expected (e.g., "daily", "hourly").
	ExpectedFrequency string `json:"expected_frequency,omitempty" yaml:"expected_frequency,omitempty"`

	// Status is "fresh", "stale", or "unknown".
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
}

// QualityRule represents a data quality rule.
type QualityRule struct {
	// Name is the rule name.
	Name string `json:"name" yaml:"name"`

	// Type is the rule type (e.g., "completeness", "uniqueness", "validity").
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Status is "passing", "failing", or "unknown".
	Status string `json:"status,omitempty" yaml:"status,omitempty"`

	// LastRun is when the rule was last evaluated.
	LastRun *time.Time `json:"last_run,omitempty" yaml:"last_run,omitempty"`
}

// TableContext contains all semantic metadata for a table.
type TableContext struct {
	// Identifier is the table identifier.
	Identifier TableIdentifier `json:"identifier" yaml:"identifier"`

	// Description is the business description of the table.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Ownership contains ownership information.
	Ownership *Ownership `json:"ownership,omitempty" yaml:"ownership,omitempty"`

	// Tags are tags attached to the table.
	Tags []Tag `json:"tags,omitempty" yaml:"tags,omitempty"`

	// GlossaryTerms are business glossary terms associated with the table.
	GlossaryTerms []GlossaryTerm `json:"glossary_terms,omitempty" yaml:"glossary_terms,omitempty"`

	// Domain is the data domain classification.
	Domain *Domain `json:"domain,omitempty" yaml:"domain,omitempty"`

	// Deprecation indicates deprecation status.
	Deprecation *Deprecation `json:"deprecation,omitempty" yaml:"deprecation,omitempty"`

	// Quality contains data quality information.
	Quality *DataQuality `json:"quality,omitempty" yaml:"quality,omitempty"`

	// CustomProperties holds provider-specific metadata.
	CustomProperties map[string]any `json:"custom_properties,omitempty" yaml:"custom_properties,omitempty"`

	// Source indicates which provider supplied this metadata.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`

	// FetchedAt is when the metadata was fetched.
	FetchedAt time.Time `json:"fetched_at,omitempty" yaml:"fetched_at,omitempty"`
}

// ColumnContext contains semantic metadata for a column.
type ColumnContext struct {
	// Identifier is the column identifier.
	Identifier ColumnIdentifier `json:"identifier" yaml:"identifier"`

	// Description is the business description of the column.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Tags are tags attached to the column.
	Tags []Tag `json:"tags,omitempty" yaml:"tags,omitempty"`

	// GlossaryTerms are business glossary terms for this column.
	GlossaryTerms []GlossaryTerm `json:"glossary_terms,omitempty" yaml:"glossary_terms,omitempty"`

	// IsSensitive indicates if the column contains sensitive data.
	IsSensitive bool `json:"is_sensitive,omitempty" yaml:"is_sensitive,omitempty"`

	// SensitivityLevel indicates the sensitivity classification.
	SensitivityLevel string `json:"sensitivity_level,omitempty" yaml:"sensitivity_level,omitempty"`

	// CustomProperties holds provider-specific metadata.
	CustomProperties map[string]any `json:"custom_properties,omitempty" yaml:"custom_properties,omitempty"`

	// Source indicates which provider supplied this metadata.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// LineageDirection specifies upstream or downstream lineage.
type LineageDirection string

const (
	// LineageUpstream traces data sources.
	LineageUpstream LineageDirection = "upstream"

	// LineageDownstream traces data consumers.
	LineageDownstream LineageDirection = "downstream"
)

// LineageInfo contains lineage information for a table.
type LineageInfo struct {
	// Table is the table this lineage is for.
	Table TableIdentifier `json:"table" yaml:"table"`

	// Direction is upstream or downstream.
	Direction LineageDirection `json:"direction" yaml:"direction"`

	// Edges are the lineage relationships.
	Edges []LineageEdge `json:"edges,omitempty" yaml:"edges,omitempty"`

	// Source indicates which provider supplied this lineage.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// LineageEdge represents a single lineage relationship.
type LineageEdge struct {
	// SourceTable is the source of the data.
	SourceTable TableIdentifier `json:"source_table" yaml:"source_table"`

	// TargetTable is the destination of the data.
	TargetTable TableIdentifier `json:"target_table" yaml:"target_table"`

	// ColumnMappings describe column-level lineage.
	ColumnMappings []ColumnMapping `json:"column_mappings,omitempty" yaml:"column_mappings,omitempty"`

	// TransformationType describes the transformation (e.g., "ETL", "view", "copy").
	TransformationType string `json:"transformation_type,omitempty" yaml:"transformation_type,omitempty"`
}

// ColumnMapping represents column-level lineage.
type ColumnMapping struct {
	// SourceColumn is the source column.
	SourceColumn string `json:"source_column" yaml:"source_column"`

	// TargetColumn is the target column.
	TargetColumn string `json:"target_column" yaml:"target_column"`

	// TransformationLogic describes how the column is transformed.
	TransformationLogic string `json:"transformation_logic,omitempty" yaml:"transformation_logic,omitempty"`
}

// SearchFilter specifies criteria for searching metadata.
type SearchFilter struct {
	// Query is a text search query.
	Query string `json:"query,omitempty" yaml:"query,omitempty"`

	// Tags filters by tag names.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Domain filters by domain URN or name.
	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`

	// Owner filters by owner ID.
	Owner string `json:"owner,omitempty" yaml:"owner,omitempty"`

	// GlossaryTerm filters by glossary term URN.
	GlossaryTerm string `json:"glossary_term,omitempty" yaml:"glossary_term,omitempty"`

	// Catalog filters by catalog name.
	Catalog string `json:"catalog,omitempty" yaml:"catalog,omitempty"`

	// Schema filters by schema name.
	Schema string `json:"schema,omitempty" yaml:"schema,omitempty"`

	// IncludeDeprecated includes deprecated assets (default: false).
	IncludeDeprecated bool `json:"include_deprecated,omitempty" yaml:"include_deprecated,omitempty"`

	// Limit is the maximum number of results to return.
	Limit int `json:"limit,omitempty" yaml:"limit,omitempty"`
}
