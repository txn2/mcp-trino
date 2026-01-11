package datahub

import (
	"fmt"
	"strings"
	"time"

	"github.com/txn2/mcp-trino/pkg/semantic"
)

// buildDatasetURN constructs a DataHub dataset URN from a table identifier.
func buildDatasetURN(table semantic.TableIdentifier, platform, environment string) string {
	// Format: urn:li:dataset:(urn:li:dataPlatform:<platform>,<catalog>.<schema>.<table>,<env>)
	qualifiedName := fmt.Sprintf("%s.%s.%s", table.Catalog, table.Schema, table.Table)
	return fmt.Sprintf("urn:li:dataset:(urn:li:dataPlatform:%s,%s,%s)", platform, qualifiedName, environment)
}

// parseDatasetURN extracts table information from a DataHub dataset URN.
func parseDatasetURN(urn string) (*semantic.TableIdentifier, error) {
	// Format: urn:li:dataset:(urn:li:dataPlatform:<platform>,<catalog>.<schema>.<table>,<env>)
	if !strings.HasPrefix(urn, "urn:li:dataset:(") {
		return nil, fmt.Errorf("invalid dataset URN: %s", urn)
	}

	// Extract the content between parentheses
	start := strings.Index(urn, "(")
	end := strings.LastIndex(urn, ")")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("invalid dataset URN format: %s", urn)
	}

	content := urn[start+1 : end]
	parts := strings.Split(content, ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid dataset URN parts: %s", urn)
	}

	// The qualified name is the second part
	qualifiedName := parts[1]
	nameParts := strings.Split(qualifiedName, ".")
	if len(nameParts) != 3 {
		return nil, fmt.Errorf("invalid qualified name: %s", qualifiedName)
	}

	return &semantic.TableIdentifier{
		Catalog: nameParts[0],
		Schema:  nameParts[1],
		Table:   nameParts[2],
	}, nil
}

// DataHub response types for JSON unmarshaling.

type datasetResponse struct {
	Dataset *datasetData `json:"dataset"`
}

type datasetData struct {
	URN           string             `json:"urn"`
	Properties    *datasetProperties `json:"properties"`
	Ownership     *ownershipData     `json:"ownership"`
	Tags          *tagsData          `json:"tags"`
	GlossaryTerms *glossaryTermsData `json:"glossaryTerms"`
	Domain        *domainData        `json:"domain"`
	Deprecation   *deprecationData   `json:"deprecation"`
}

type datasetProperties struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ownershipData struct {
	Owners []ownerEntry `json:"owners"`
}

type ownerEntry struct {
	Owner         ownerInfo `json:"owner"`
	OwnershipType *struct {
		URN  string `json:"urn"`
		Info *struct {
			Name string `json:"name"`
		} `json:"info"`
	} `json:"ownershipType"`
}

type ownerInfo struct {
	URN        string `json:"urn"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Properties *struct {
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
	} `json:"properties"`
}

type tagsData struct {
	Tags []tagEntry `json:"tags"`
}

type tagEntry struct {
	Tag struct {
		URN        string `json:"urn"`
		Properties *struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"properties"`
	} `json:"tag"`
}

type glossaryTermsData struct {
	Terms []glossaryTermEntry `json:"terms"`
}

type glossaryTermEntry struct {
	Term struct {
		URN        string `json:"urn"`
		Properties *struct {
			Name       string `json:"name"`
			Definition string `json:"definition"`
		} `json:"properties"`
	} `json:"term"`
}

type domainData struct {
	Domain *struct {
		URN        string `json:"urn"`
		Properties *struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"properties"`
	} `json:"domain"`
}

type deprecationData struct {
	Deprecated       bool   `json:"deprecated"`
	Note             string `json:"note"`
	DecommissionTime int64  `json:"decommissionTime"`
}

type schemaResponse struct {
	Dataset *struct {
		SchemaMetadata *schemaMetadata `json:"schemaMetadata"`
	} `json:"dataset"`
}

type schemaMetadata struct {
	Fields []fieldEntry `json:"fields"`
}

type fieldEntry struct {
	FieldPath     string             `json:"fieldPath"`
	Description   string             `json:"description"`
	Tags          *tagsData          `json:"tags"`
	GlossaryTerms *glossaryTermsData `json:"glossaryTerms"`
}

type glossaryTermResponse struct {
	GlossaryTerm *glossaryTermData `json:"glossaryTerm"`
}

type glossaryTermData struct {
	URN        string `json:"urn"`
	Properties *struct {
		Name       string `json:"name"`
		Definition string `json:"definition"`
		TermSource string `json:"termSource"`
	} `json:"properties"`
	RelatedTerms *struct {
		Terms []struct {
			Term struct {
				URN string `json:"urn"`
			} `json:"term"`
		} `json:"terms"`
	} `json:"relatedTerms"`
}

type searchResponse struct {
	Search struct {
		SearchResults []searchResult `json:"searchResults"`
	} `json:"search"`
}

type searchResult struct {
	Entity struct {
		URN        string `json:"urn"`
		Name       string `json:"name"`
		Properties *struct {
			Name          string `json:"name"`
			QualifiedName string `json:"qualifiedName"`
		} `json:"properties"`
		Platform *struct {
			Name string `json:"name"`
		} `json:"platform"`
	} `json:"entity"`
}

// mapDatasetToTableContext converts DataHub dataset data to TableContext.
func mapDatasetToTableContext(data *datasetData, table semantic.TableIdentifier) *semantic.TableContext {
	if data == nil {
		return nil
	}

	ctx := &semantic.TableContext{
		Identifier: table,
		Source:     ProviderName,
		FetchedAt:  time.Now(),
	}

	// Map properties
	if data.Properties != nil {
		ctx.Description = data.Properties.Description
	}

	// Map ownership, tags, glossary terms, domain, and deprecation
	ctx.Ownership = mapOwnership(data.Ownership)
	ctx.Tags = mapTags(data.Tags)
	ctx.GlossaryTerms = mapGlossaryTerms(data.GlossaryTerms)
	ctx.Domain = mapDomain(data.Domain)
	ctx.Deprecation = mapDeprecation(data.Deprecation)

	return ctx
}

// mapOwnership converts DataHub ownership data to semantic.Ownership.
func mapOwnership(data *ownershipData) *semantic.Ownership {
	if data == nil || len(data.Owners) == 0 {
		return nil
	}

	owners := make([]semantic.Owner, 0, len(data.Owners))
	for _, o := range data.Owners {
		owner := mapOwnerEntry(o)
		owners = append(owners, owner)
	}
	return &semantic.Ownership{Owners: owners}
}

// mapOwnerEntry converts a single owner entry to semantic.Owner.
func mapOwnerEntry(o ownerEntry) semantic.Owner {
	owner := semantic.Owner{
		ID: o.Owner.URN,
	}

	// Determine type and name
	if o.Owner.Username != "" {
		owner.Type = "user"
		owner.Name = o.Owner.Username
		if o.Owner.Properties != nil && o.Owner.Properties.DisplayName != "" {
			owner.Name = o.Owner.Properties.DisplayName
		}
	} else if o.Owner.Name != "" {
		owner.Type = "group"
		owner.Name = o.Owner.Name
		if o.Owner.Properties != nil && o.Owner.Properties.DisplayName != "" {
			owner.Name = o.Owner.Properties.DisplayName
		}
	}

	// Get role from ownership type
	if o.OwnershipType != nil && o.OwnershipType.Info != nil {
		owner.Role = o.OwnershipType.Info.Name
	}

	return owner
}

// mapTags converts DataHub tags data to semantic.Tag slice.
func mapTags(data *tagsData) []semantic.Tag {
	if data == nil || len(data.Tags) == 0 {
		return nil
	}

	tags := make([]semantic.Tag, 0, len(data.Tags))
	for _, t := range data.Tags {
		if t.Tag.Properties != nil {
			tags = append(tags, semantic.Tag{
				Name:        t.Tag.Properties.Name,
				Description: t.Tag.Properties.Description,
				Source:      ProviderName,
			})
		}
	}
	return tags
}

// mapGlossaryTerms converts DataHub glossary terms data to semantic.GlossaryTerm slice.
func mapGlossaryTerms(data *glossaryTermsData) []semantic.GlossaryTerm {
	if data == nil || len(data.Terms) == 0 {
		return nil
	}

	terms := make([]semantic.GlossaryTerm, 0, len(data.Terms))
	for _, t := range data.Terms {
		term := semantic.GlossaryTerm{
			URN:    t.Term.URN,
			Source: ProviderName,
		}
		if t.Term.Properties != nil {
			term.Name = t.Term.Properties.Name
			term.Definition = t.Term.Properties.Definition
		}
		terms = append(terms, term)
	}
	return terms
}

// mapDomain converts DataHub domain data to semantic.Domain.
func mapDomain(data *domainData) *semantic.Domain {
	if data == nil || data.Domain == nil {
		return nil
	}

	domain := &semantic.Domain{
		URN: data.Domain.URN,
	}
	if data.Domain.Properties != nil {
		domain.Name = data.Domain.Properties.Name
		domain.Description = data.Domain.Properties.Description
	}
	return domain
}

// mapDeprecation converts DataHub deprecation data to semantic.Deprecation.
func mapDeprecation(data *deprecationData) *semantic.Deprecation {
	if data == nil || !data.Deprecated {
		return nil
	}

	deprecation := &semantic.Deprecation{
		Deprecated: true,
		Note:       data.Note,
	}
	if data.DecommissionTime > 0 {
		t := time.UnixMilli(data.DecommissionTime)
		deprecation.DecommissionTime = &t
	}
	return deprecation
}

// mapSchemaToColumnContexts converts DataHub schema data to column contexts.
func mapSchemaToColumnContexts(data *schemaMetadata, table semantic.TableIdentifier) map[string]*semantic.ColumnContext {
	if data == nil || len(data.Fields) == 0 {
		return nil
	}

	result := make(map[string]*semantic.ColumnContext, len(data.Fields))
	for _, field := range data.Fields {
		ctx := mapFieldToColumnContext(field, table)
		columnName := extractColumnName(field.FieldPath)
		result[columnName] = ctx
	}

	return result
}

// extractColumnName extracts the column name from a field path.
func extractColumnName(fieldPath string) string {
	if idx := strings.LastIndex(fieldPath, "."); idx != -1 {
		return fieldPath[idx+1:]
	}
	return fieldPath
}

// mapFieldToColumnContext converts a single field entry to ColumnContext.
func mapFieldToColumnContext(field fieldEntry, table semantic.TableIdentifier) *semantic.ColumnContext {
	columnName := extractColumnName(field.FieldPath)
	tags, isSensitive, sensitivityLevel := mapColumnTagsWithSensitivity(field.Tags)

	return &semantic.ColumnContext{
		Identifier: semantic.ColumnIdentifier{
			TableIdentifier: table,
			Column:          columnName,
		},
		Description:      field.Description,
		Source:           ProviderName,
		Tags:             tags,
		GlossaryTerms:    mapGlossaryTerms(field.GlossaryTerms),
		IsSensitive:      isSensitive,
		SensitivityLevel: sensitivityLevel,
	}
}

// mapColumnTagsWithSensitivity maps column tags and detects sensitivity markers.
func mapColumnTagsWithSensitivity(data *tagsData) (tags []semantic.Tag, isSensitive bool, sensitivityLevel string) {
	if data == nil || len(data.Tags) == 0 {
		return nil, false, ""
	}

	tags = make([]semantic.Tag, 0, len(data.Tags))
	for _, t := range data.Tags {
		if t.Tag.Properties != nil {
			tag := semantic.Tag{
				Name:   t.Tag.Properties.Name,
				Source: ProviderName,
			}
			// Check for sensitivity tags
			tagLower := strings.ToLower(t.Tag.Properties.Name)
			if strings.Contains(tagLower, "pii") || strings.Contains(tagLower, "sensitive") {
				isSensitive = true
				sensitivityLevel = t.Tag.Properties.Name
			}
			tags = append(tags, tag)
		}
	}

	return tags, isSensitive, sensitivityLevel
}

// mapGlossaryTermData converts DataHub glossary term data to GlossaryTerm.
func mapGlossaryTermData(data *glossaryTermData) *semantic.GlossaryTerm {
	if data == nil {
		return nil
	}

	term := &semantic.GlossaryTerm{
		URN:    data.URN,
		Source: ProviderName,
	}

	if data.Properties != nil {
		term.Name = data.Properties.Name
		term.Definition = data.Properties.Definition
	}

	if data.RelatedTerms != nil && len(data.RelatedTerms.Terms) > 0 {
		term.RelatedTerms = make([]string, 0, len(data.RelatedTerms.Terms))
		for _, t := range data.RelatedTerms.Terms {
			term.RelatedTerms = append(term.RelatedTerms, t.Term.URN)
		}
	}

	return term
}
