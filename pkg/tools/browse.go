package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BrowseInput defines the input for the trino_browse tool.
// The browsing level is determined by which parameters are provided:
//   - No catalog, no schema → list catalogs
//   - Catalog only → list schemas in that catalog
//   - Catalog + schema → list tables in that schema
type BrowseInput struct {
	// Catalog is the catalog name. Omit to list all catalogs.
	Catalog string `json:"catalog,omitempty" jsonschema_description:"Catalog name. Omit to list all catalogs."`

	// Schema is the schema name. Requires catalog. Omit to list schemas.
	Schema string `json:"schema,omitempty" jsonschema_description:"Schema name. Requires catalog. Omit to list schemas."`

	// Pattern is an optional LIKE pattern to filter tables (only when listing tables).
	Pattern string `json:"pattern,omitempty" jsonschema_description:"LIKE pattern to filter tables (only when listing tables)"`

	// Connection is the named connection to use. Empty uses the default connection.
	// Use trino_list_connections to see available connections.
	Connection string `json:"connection,omitempty" jsonschema_description:"Named connection to use (see trino_list_connections)"`
}

// registerBrowseTool adds the trino_browse tool to the server.
//
//nolint:dupl // Each tool registration requires distinct types for type-safe handlers.
func (t *Toolkit) registerBrowseTool(server *mcp.Server, cfg *toolConfig) {
	baseHandler := func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		browseInput, ok := input.(BrowseInput)
		if !ok {
			return ErrorResult("internal error: invalid input type"), nil, nil
		}
		return t.handleBrowse(ctx, req, browseInput)
	}

	wrappedHandler := t.wrapHandler(ToolBrowse, baseHandler, cfg)

	mcp.AddTool(server, &mcp.Tool{
		Name:        string(ToolBrowse),
		Title:       t.getTitle(ToolBrowse, cfg),
		Description: t.getDescription(ToolBrowse, cfg),
		Annotations: t.getAnnotations(ToolBrowse, cfg),
		Icons:       t.getIcons(ToolBrowse, cfg),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input BrowseInput) (*mcp.CallToolResult, *BrowseOutput, error) {
		result, out, err := wrappedHandler(ctx, req, input)
		if typed, ok := out.(*BrowseOutput); ok {
			return result, typed, err
		}
		return result, nil, err
	})
}

func (t *Toolkit) handleBrowse(
	ctx context.Context, _ *mcp.CallToolRequest, input BrowseInput,
) (*mcp.CallToolResult, any, error) {
	if err := validateBrowseInput(input); err != nil {
		return ErrorResult(err.Error()), nil, nil
	}

	trinoClient, err := t.getClient(input.Connection)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Connection error: %v", err)), nil, nil
	}

	switch {
	case input.Catalog == "":
		return t.browseCatalogs(ctx, trinoClient)
	case input.Schema == "":
		return t.browseSchemas(ctx, trinoClient, input.Catalog)
	default:
		return t.browseTables(ctx, trinoClient, input.Catalog, input.Schema, input.Pattern)
	}
}

func validateBrowseInput(input BrowseInput) error {
	if input.Schema != "" && input.Catalog == "" {
		return fmt.Errorf("schema requires catalog")
	}
	if input.Pattern != "" && (input.Catalog == "" || input.Schema == "") {
		return fmt.Errorf("pattern requires both catalog and schema")
	}
	return nil
}

func (t *Toolkit) browseCatalogs(
	ctx context.Context, trinoClient TrinoClient,
) (*mcp.CallToolResult, any, error) {
	catalogs, err := trinoClient.ListCatalogs(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list catalogs: %v", err)), nil, nil
	}

	output := "## Available Catalogs\n\n"
	for _, catalog := range catalogs {
		output += fmt.Sprintf("- `%s`\n", catalog)
	}
	output += fmt.Sprintf("\n*%d catalogs found*", len(catalogs))

	browseOutput := BrowseOutput{
		Level: "catalogs",
		Items: catalogs,
		Count: len(catalogs),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, &browseOutput, nil
}

func (t *Toolkit) browseSchemas(
	ctx context.Context, trinoClient TrinoClient, catalog string,
) (*mcp.CallToolResult, any, error) {
	schemas, err := trinoClient.ListSchemas(ctx, catalog)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list schemas: %v", err)), nil, nil
	}

	output := fmt.Sprintf("## Schemas in `%s`\n\n", catalog)
	for _, schema := range schemas {
		output += fmt.Sprintf("- `%s`\n", schema)
	}
	output += fmt.Sprintf("\n*%d schemas found*", len(schemas))

	browseOutput := BrowseOutput{
		Level:   "schemas",
		Catalog: catalog,
		Items:   schemas,
		Count:   len(schemas),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, &browseOutput, nil
}

func (t *Toolkit) browseTables(
	ctx context.Context, trinoClient TrinoClient, catalog, schema, pattern string,
) (*mcp.CallToolResult, any, error) {
	tables, err := trinoClient.ListTables(ctx, catalog, schema)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list tables: %v", err)), nil, nil
	}

	var tableNames []string
	var output string

	if pattern != "" {
		p := strings.ToLower(pattern)
		p = strings.ReplaceAll(p, "%", "")
		for _, tbl := range tables {
			if strings.Contains(strings.ToLower(tbl.Name), p) {
				tableNames = append(tableNames, tbl.Name)
			}
		}
		output = fmt.Sprintf("## Tables in `%s.%s` matching '%s'\n\n", catalog, schema, pattern)
	} else {
		tableNames = make([]string, len(tables))
		for i, tbl := range tables {
			tableNames[i] = tbl.Name
		}
		output = fmt.Sprintf("## Tables in `%s.%s`\n\n", catalog, schema)
	}

	for _, name := range tableNames {
		output += fmt.Sprintf("- `%s`\n", name)
	}
	output += fmt.Sprintf("\n*%d tables found*", len(tableNames))

	browseOutput := BrowseOutput{
		Level:   "tables",
		Catalog: catalog,
		Schema:  schema,
		Items:   tableNames,
		Count:   len(tableNames),
		Pattern: pattern,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, &browseOutput, nil
}
