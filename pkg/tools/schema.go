package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/pkg/client"
)

// ListCatalogsInput defines the input for the trino_list_catalogs tool.
type ListCatalogsInput struct {
	// No input required
}

// registerListCatalogsTool adds the trino_list_catalogs tool to the server.
func (t *Toolkit) registerListCatalogsTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_list_catalogs",
		Description: "List all available catalogs in the Trino cluster. Catalogs are the top-level containers for schemas and tables.",
	}, t.handleListCatalogs)
}

func (t *Toolkit) handleListCatalogs(ctx context.Context, _ *mcp.CallToolRequest, _ ListCatalogsInput) (*mcp.CallToolResult, any, error) {
	catalogs, err := t.client.ListCatalogs(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to list catalogs: %v", err)), nil, nil
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
}

// registerListSchemasTool adds the trino_list_schemas tool to the server.
func (t *Toolkit) registerListSchemasTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_list_schemas",
		Description: "List all schemas in a catalog. Schemas are containers for tables within a catalog.",
	}, t.handleListSchemas)
}

func (t *Toolkit) handleListSchemas(ctx context.Context, _ *mcp.CallToolRequest, input ListSchemasInput) (*mcp.CallToolResult, any, error) {
	if input.Catalog == "" {
		return errorResult("catalog parameter is required"), nil, nil
	}

	schemas, err := t.client.ListSchemas(ctx, input.Catalog)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to list schemas: %v", err)), nil, nil
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
}

// registerListTablesTool adds the trino_list_tables tool to the server.
func (t *Toolkit) registerListTablesTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_list_tables",
		Description: "List all tables in a schema. Optionally filter by a LIKE pattern.",
	}, t.handleListTables)
}

func (t *Toolkit) handleListTables(ctx context.Context, _ *mcp.CallToolRequest, input ListTablesInput) (*mcp.CallToolResult, any, error) {
	if input.Catalog == "" {
		return errorResult("catalog parameter is required"), nil, nil
	}
	if input.Schema == "" {
		return errorResult("schema parameter is required"), nil, nil
	}

	tables, err := t.client.ListTables(ctx, input.Catalog, input.Schema)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to list tables: %v", err)), nil, nil
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
}

// registerDescribeTableTool adds the trino_describe_table tool to the server.
func (t *Toolkit) registerDescribeTableTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trino_describe_table",
		Description: "Get detailed information about a table including column names, types, and optionally a sample of data.",
	}, t.handleDescribeTable)
}

func (t *Toolkit) handleDescribeTable(
	ctx context.Context, _ *mcp.CallToolRequest, input DescribeTableInput,
) (*mcp.CallToolResult, any, error) {
	if input.Catalog == "" {
		return errorResult("catalog parameter is required"), nil, nil
	}
	if input.Schema == "" {
		return errorResult("schema parameter is required"), nil, nil
	}
	if input.Table == "" {
		return errorResult("table parameter is required"), nil, nil
	}

	info, err := t.client.DescribeTable(ctx, input.Catalog, input.Schema, input.Table)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to describe table: %v", err)), nil, nil
	}

	output := fmt.Sprintf("## Table: `%s.%s.%s`\n\n", info.Catalog, info.Schema, info.Name)
	output += "### Columns\n\n"
	output += "| Name | Type | Nullable | Comment |\n"
	output += "|------|------|----------|----------|\n"

	for _, col := range info.Columns {
		nullable := col.Nullable
		if nullable == "" {
			nullable = "-"
		}
		comment := col.Comment
		if comment == "" {
			comment = "-"
		}
		output += fmt.Sprintf("| `%s` | %s | %s | %s |\n", col.Name, col.Type, nullable, comment)
	}

	output += fmt.Sprintf("\n*%d columns*", len(info.Columns))

	// Include sample if requested
	if input.IncludeSample {
		sampleSQL := fmt.Sprintf("SELECT * FROM %s.%s.%s LIMIT 5", input.Catalog, input.Schema, input.Table)
		sampleOpts := client.DefaultQueryOptions()
		sampleOpts.Limit = 5

		sample, err := t.client.Query(ctx, sampleSQL, sampleOpts)
		if err == nil && len(sample.Rows) > 0 {
			output += "\n\n### Sample Data\n\n"
			sampleJSON, err := json.MarshalIndent(sample.Rows, "", "  ")
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to marshal sample: %v", err)), nil, nil
			}
			output += "```json\n" + string(sampleJSON) + "\n```"
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil, nil
}
