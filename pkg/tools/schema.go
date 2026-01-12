package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
	"github.com/txn2/mcp-trino/pkg/semantic"
)

// ListCatalogsInput defines the input for the trino_list_catalogs tool.
type ListCatalogsInput struct {
	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerListCatalogsTool adds the trino_list_catalogs tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerListCatalogsTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		listInput, ok := input.(ListCatalogsInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleListCatalogs(ctx, req, listInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolListCatalogs, baseHandler, cfg)

	// Register with MCP
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_list_catalogs",
		Description: "List all available catalogs in the Trino cluster. Catalogs are the top-level containers for schemas and tables.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListCatalogsInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

func (t *Toolkit) handleListCatalogs(
	ctx context.Context, _ *mcp.CallToolRequest, input ListCatalogsInput,
) (*mcp.CallToolResult, any, error) {
	// Get client for the specified connection
	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	catalogs, err := trinoClient.ListCatalogs(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list catalogs: %v", err)), nil, nil
	}

	output := "## Available Catalogs\n\n"
	for _, catalog := range catalogs {
		output += fmt.Sprintf("- `%s`\n", catalog)
	}
	output += fmt.Sprintf("\n*%d catalogs found*", len(catalogs))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

// ListSchemasInput defines the input for the trino_list_schemas tool.
type ListSchemasInput struct {
	// Catalog is the catalog to list schemas from.
	Catalog string `json:"catalog" jsonschema_description:"The catalog to list schemas from"`

	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerListSchemasTool adds the trino_list_schemas tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerListSchemasTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		schemaInput, ok := input.(ListSchemasInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleListSchemas(ctx, req, schemaInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolListSchemas, baseHandler, cfg)

	// Register with MCP
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_list_schemas",
		Description: "List all schemas in a catalog. Schemas are containers for tables within a catalog.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListSchemasInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

func (t *Toolkit) handleListSchemas(ctx context.Context, _ *mcp.CallToolRequest, input ListSchemasInput) (*mcp.CallToolResult, any, error) {
	if input.Catalog == "" {
		return ErrorResult("catalog parameter is required"), nil, nil
	}

	// Get client for the specified connection
	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	schemas, err := trinoClient.ListSchemas(ctx, input.Catalog)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list schemas: %v", err)), nil, nil
	}

	output := fmt.Sprintf("## Schemas in `%s`\n\n", input.Catalog)
	for _, schema := range schemas {
		output += fmt.Sprintf("- `%s`\n", schema)
	}
	output += fmt.Sprintf("\n*%d schemas found*", len(schemas))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

// ListTablesInput defines the input for the trino_list_tables tool.
type ListTablesInput struct {
	// Catalog is the catalog containing the schema.
	Catalog string `json:"catalog" jsonschema_description:"The catalog containing the schema"`

	// Schema is the schema to list tables from.
	Schema string `json:"schema" jsonschema_description:"The schema to list tables from"`

	// Pattern is an optional LIKE pattern to filter table names.
	Pattern string `json:"pattern,omitempty" jsonschema_description:"Optional LIKE pattern to filter tables (e.g., '%user%')"`

	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerListTablesTool adds the trino_list_tables tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerListTablesTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		tablesInput, ok := input.(ListTablesInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleListTables(ctx, req, tablesInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolListTables, baseHandler, cfg)

	// Register with MCP
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_list_tables",
		Description: "List all tables in a schema. Optionally filter by a LIKE pattern.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTablesInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

func (t *Toolkit) handleListTables(ctx context.Context, _ *mcp.CallToolRequest, input ListTablesInput) (*mcp.CallToolResult, any, error) {
	if input.Catalog == "" {
		return ErrorResult("catalog parameter is required"), nil, nil
	}
	if input.Schema == "" {
		return ErrorResult("schema parameter is required"), nil, nil
	}

	// Get client for the specified connection
	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	tables, err := trinoClient.ListTables(ctx, input.Catalog, input.Schema)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list tables: %v", err)), nil, nil
	}

	// Filter by pattern if provided
	if input.Pattern != "" {
		pattern := strings.ToLower(input.Pattern)
		pattern = strings.ReplaceAll(pattern, "%", "")
		filtered := make([]string, 0)
		for _, tbl := range tables {
			if strings.Contains(strings.ToLower(tbl.Name), pattern) {
				filtered = append(filtered, tbl.Name)
			}
		}
		output := fmt.Sprintf("## Tables in `%s.%s` matching '%s'\n\n", input.Catalog, input.Schema, input.Pattern)
		for _, name := range filtered {
			output += fmt.Sprintf("- `%s`\n", name)
		}
		output += fmt.Sprintf("\n*%d tables found*", len(filtered))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil, nil
	}

	output := fmt.Sprintf("## Tables in `%s.%s`\n\n", input.Catalog, input.Schema)
	for _, tbl := range tables {
		output += fmt.Sprintf("- `%s`\n", tbl.Name)
	}
	output += fmt.Sprintf("\n*%d tables found*", len(tables))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

// DescribeTableInput defines the input for the trino_describe_table tool.
type DescribeTableInput struct {
	// Catalog is the catalog containing the table.
	Catalog string `json:"catalog" jsonschema_description:"The catalog containing the table"`

	// Schema is the schema containing the table.
	Schema string `json:"schema" jsonschema_description:"The schema containing the table"`

	// Table is the table name to describe.
	Table string `json:"table" jsonschema_description:"The table name to describe"`

	// IncludeSample includes a sample of data rows.
	IncludeSample bool `json:"include_sample,omitempty" jsonschema_description:"Include a 5-row sample of data"`

	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerDescribeTableTool adds the trino_describe_table tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerDescribeTableTool(server *mcp.Server, cfg *toolConfig) {
	// Create the base handler
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		describeInput, ok := input.(DescribeTableInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleDescribeTable(ctx, req, describeInput)
	}

	// Wrap with middleware if configured
	wrappedHandler := t.wrapHandler(ToolDescribeTable, baseHandler, cfg)

	// Register with MCP
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_describe_table",
		Description: "Get detailed information about a table including column names, types, and optionally a sample of data.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DescribeTableInput) (*mcp.CallToolResult, any, error) {
		return wrappedHandler(ctx, req, input)
	})
}

func (t *Toolkit) handleDescribeTable(
	ctx context.Context, _ *mcp.CallToolRequest, input DescribeTableInput,
) (*mcp.CallToolResult, any, error) {
	if err := validateDescribeTableInput(input); err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	info, err := trinoClient.DescribeTable(ctx, input.Catalog, input.Schema, input.Table)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to describe table: %v", err)), nil, nil
	}

	output := fmt.Sprintf("## Table: `%s.%s.%s`\n\n", info.Catalog, info.Schema, info.Name)
	output += t.formatTableWithSemantics(ctx, input, info)

	if input.IncludeSample {
		sampleOutput, err := t.formatSampleData(ctx, trinoClient, input)
		if err != nil {
			return ErrorResult(err.Error()), nil, nil
		}
		output += sampleOutput
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}

func validateDescribeTableInput(input DescribeTableInput) error {
	if input.Catalog == "" {
		return fmt.Errorf("catalog parameter is required")
	}
	if input.Schema == "" {
		return fmt.Errorf("schema parameter is required")
	}
	if input.Table == "" {
		return fmt.Errorf("table parameter is required")
	}
	return nil
}

func (t *Toolkit) formatTableWithSemantics(
	ctx context.Context, input DescribeTableInput, info *client.TableInfo,
) string {
	var output string
	var columnSemantics map[string]*semantic.ColumnContext

	if t.semanticProvider != nil {
		tableID := semantic.TableIdentifier{
			Connection: input.Connection,
			Catalog:    input.Catalog,
			Schema:     input.Schema,
			Table:      input.Table,
		}
		output += t.enrichWithTableContext(ctx, tableID)
		//nolint:errcheck // semantic enrichment is optional
		columnSemantics, _ = t.semanticProvider.GetColumnsContext(ctx, tableID)
	}

	output += t.formatColumns(info.Columns, columnSemantics)
	output += fmt.Sprintf("\n*%d columns*", len(info.Columns))
	return output
}

func (t *Toolkit) enrichWithTableContext(ctx context.Context, tableID semantic.TableIdentifier) string {
	tableCtx, err := t.semanticProvider.GetTableContext(ctx, tableID)
	if err != nil || tableCtx == nil {
		return ""
	}
	return t.formatTableSemantics(tableCtx)
}

func (t *Toolkit) formatColumns(columns []client.ColumnDef, semantics map[string]*semantic.ColumnContext) string {
	if len(semantics) > 0 {
		return t.formatColumnsWithSemantics(columns, semantics)
	}
	return formatBasicColumns(columns)
}

func formatBasicColumns(columns []client.ColumnDef) string {
	var sb strings.Builder
	sb.WriteString("### Columns\n\n")
	sb.WriteString("| Name | Type | Nullable | Comment |\n")
	sb.WriteString("|------|------|----------|----------|\n")

	for _, col := range columns {
		nullable := col.Nullable
		if nullable == "" {
			nullable = "-"
		}
		comment := col.Comment
		if comment == "" {
			comment = "-"
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", col.Name, col.Type, nullable, comment))
	}
	return sb.String()
}

func (t *Toolkit) formatSampleData(
	ctx context.Context, trinoClient TrinoClient, input DescribeTableInput,
) (string, error) {
	sampleSQL := fmt.Sprintf("SELECT * FROM %s.%s.%s LIMIT 5", input.Catalog, input.Schema, input.Table)
	sampleOpts := client.DefaultQueryOptions()
	sampleOpts.Limit = 5

	sample, err := trinoClient.Query(ctx, sampleSQL, sampleOpts)
	if err != nil || len(sample.Rows) == 0 {
		return "", nil
	}

	sampleJSON, err := json.MarshalIndent(sample.Rows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sample: %w", err)
	}
	return "\n\n### Sample Data\n\n```json\n" + string(sampleJSON) + "\n```", nil
}

// formatTableSemantics formats semantic metadata for a table.
func (t *Toolkit) formatTableSemantics(tc *semantic.TableContext) string {
	var sb strings.Builder
	sb.WriteString(formatDescription(tc.Description))
	sb.WriteString(formatDeprecation(tc.Deprecation))
	sb.WriteString(formatOwnership(tc.Ownership))
	sb.WriteString(formatTags(tc.Tags))
	sb.WriteString(formatDomain(tc.Domain))
	sb.WriteString(formatGlossaryTerms(tc.GlossaryTerms))
	sb.WriteString(formatQuality(tc.Quality))
	return sb.String()
}

func formatDescription(desc string) string {
	if desc == "" {
		return ""
	}
	return fmt.Sprintf("**Description:** %s\n\n", desc)
}

func formatDeprecation(d *semantic.Deprecation) string {
	if d == nil || !d.Deprecated {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("> **DEPRECATED:** %s\n", d.Note))
	if d.ReplacedBy != "" {
		sb.WriteString(fmt.Sprintf("> Use `%s` instead.\n", d.ReplacedBy))
	}
	sb.WriteString("\n")
	return sb.String()
}

func formatOwnership(o *semantic.Ownership) string {
	if o == nil || len(o.Owners) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("**Owners:** ")
	for i, owner := range o.Owners {
		if i > 0 {
			sb.WriteString(", ")
		}
		if owner.Role != "" {
			sb.WriteString(fmt.Sprintf("%s (%s)", owner.Name, owner.Role))
		} else {
			sb.WriteString(owner.Name)
		}
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func formatTags(tags []semantic.Tag) string {
	if len(tags) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("**Tags:** ")
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("`%s`", tag.Name))
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func formatDomain(d *semantic.Domain) string {
	if d == nil || d.Name == "" {
		return ""
	}
	return fmt.Sprintf("**Domain:** %s\n\n", d.Name)
}

func formatGlossaryTerms(terms []semantic.GlossaryTerm) string {
	if len(terms) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("**Glossary Terms:** ")
	for i, term := range terms {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("*%s*", term.Name))
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func formatQuality(q *semantic.DataQuality) string {
	if q == nil || q.Score == nil {
		return ""
	}
	return fmt.Sprintf("**Data Quality Score:** %.0f%%\n\n", *q.Score)
}

// formatColumnsWithSemantics formats columns with semantic enrichment.
func (t *Toolkit) formatColumnsWithSemantics(columns []client.ColumnDef, semantics map[string]*semantic.ColumnContext) string {
	var sb strings.Builder

	sb.WriteString("### Columns\n\n")
	sb.WriteString("| Name | Type | Nullable | Description | Tags |\n")
	sb.WriteString("|------|------|----------|-------------|------|\n")

	for _, col := range columns {
		nullable := col.Nullable
		if nullable == "" {
			nullable = "-"
		}

		description := col.Comment
		tags := "-"

		// Enrich with semantic metadata
		if colCtx, ok := semantics[col.Name]; ok && colCtx != nil {
			if colCtx.Description != "" {
				description = colCtx.Description
			}
			if len(colCtx.Tags) > 0 {
				tagNames := make([]string, len(colCtx.Tags))
				for i, tag := range colCtx.Tags {
					tagNames[i] = tag.Name
				}
				tags = strings.Join(tagNames, ", ")
			}
			if colCtx.IsSensitive {
				if tags == "-" {
					tags = "**SENSITIVE**"
				} else {
					tags += ", **SENSITIVE**"
				}
			}
		}

		if description == "" {
			description = "-"
		}

		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
			col.Name, col.Type, nullable, description, tags))
	}

	return sb.String()
}
