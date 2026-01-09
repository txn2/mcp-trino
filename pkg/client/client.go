package client

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/trinodb/trino-go-client/trino" // Trino database driver
)

// Client is a wrapper around the Trino database connection.
type Client struct {
	db     *sql.DB
	config Config
}

// New creates a new Trino client with the given configuration.
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	db, err := sql.Open("trino", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// NewWithDB creates a new client with an existing database connection.
// This is primarily useful for testing with mock databases.
func NewWithDB(db *sql.DB, cfg Config) *Client {
	return &Client{
		db:     db,
		config: cfg,
	}
}

// Close closes the database connection.
func (c *Client) Close() error {
	return c.db.Close()
}

// Ping tests the connection to Trino.
func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Config returns the client configuration.
func (c *Client) Config() Config {
	return c.config
}

// QueryResult represents the result of a SQL query.
type QueryResult struct {
	Columns []ColumnInfo     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Stats   QueryStats       `json:"stats"`
}

// ColumnInfo describes a column in the result set.
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// QueryStats contains execution statistics.
type QueryStats struct {
	RowCount     int           `json:"row_count"`
	Duration     time.Duration `json:"duration_ms"`
	Truncated    bool          `json:"truncated"`
	LimitApplied int           `json:"limit_applied,omitempty"`
}

// QueryOptions configures query execution.
type QueryOptions struct {
	// Limit is the maximum number of rows to return. Default: 1000.
	Limit int

	// Timeout is the query timeout. Uses client default if not set.
	Timeout time.Duration

	// Catalog overrides the default catalog for this query.
	Catalog string

	// Schema overrides the default schema for this query.
	Schema string
}

// DefaultQueryOptions returns default query options.
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		Limit: 1000,
	}
}

// Query executes a SQL query and returns the results.
func (c *Client) Query(ctx context.Context, sql string, opts QueryOptions) (*QueryResult, error) {
	start := time.Now()

	// Apply timeout
	timeout := c.config.Timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Apply limit if not already in query
	limit := opts.Limit
	if limit <= 0 {
		limit = 1000
	}

	// Execute query
	rows, err := c.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column info
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	columns := make([]ColumnInfo, len(columnTypes))
	for i, ct := range columnTypes {
		nullable, _ := ct.Nullable()
		columns[i] = ColumnInfo{
			Name:     ct.Name(),
			Type:     ct.DatabaseTypeName(),
			Nullable: nullable,
		}
	}

	// Scan rows
	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]any, 0),
	}

	rowCount := 0
	truncated := false

	for rows.Next() {
		if rowCount >= limit {
			truncated = true
			break
		}

		// Create scan destinations
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]any)
		for i, col := range columns {
			row[col.Name] = convertValue(values[i])
		}
		result.Rows = append(result.Rows, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	result.Stats = QueryStats{
		RowCount:     rowCount,
		Duration:     time.Since(start),
		Truncated:    truncated,
		LimitApplied: limit,
	}

	return result, nil
}

// ExplainType specifies the type of EXPLAIN output.
type ExplainType string

const (
	ExplainLogical     ExplainType = "LOGICAL"
	ExplainDistributed ExplainType = "DISTRIBUTED"
	ExplainIO          ExplainType = "(TYPE IO)"
	ExplainValidate    ExplainType = "(TYPE VALIDATE)"
)

// ExplainResult holds the output of an EXPLAIN query.
type ExplainResult struct {
	Type ExplainType `json:"type"`
	Plan string      `json:"plan"`
}

// Explain returns the execution plan for a query.
func (c *Client) Explain(ctx context.Context, sql string, explainType ExplainType) (*ExplainResult, error) {
	if explainType == "" {
		explainType = ExplainLogical
	}

	var explainSQL string
	switch explainType {
	case ExplainIO, ExplainValidate:
		explainSQL = fmt.Sprintf("EXPLAIN %s %s", explainType, sql) // #nosec G201 -- explainType is from enum, sql is validated
	default:
		explainSQL = fmt.Sprintf("EXPLAIN (%s) %s", explainType, sql) // #nosec G201 -- explainType is from enum, sql is validated
	}

	rows, err := c.db.QueryContext(ctx, explainSQL)
	if err != nil {
		return nil, fmt.Errorf("explain failed: %w", err)
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, fmt.Errorf("failed to scan explain output: %w", err)
		}
		planLines = append(planLines, line)
	}

	return &ExplainResult{
		Type: explainType,
		Plan: strings.Join(planLines, "\n"),
	}, nil
}

// TableInfo holds metadata about a table.
type TableInfo struct {
	Catalog string      `json:"catalog"`
	Schema  string      `json:"schema"`
	Name    string      `json:"name"`
	Type    string      `json:"type"` // TABLE, VIEW, etc.
	Columns []ColumnDef `json:"columns,omitempty"`
}

// ColumnDef describes a column definition.
type ColumnDef struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
	Comment  string `json:"comment,omitempty"`
}

// ListCatalogs returns available catalogs.
func (c *Client) ListCatalogs(ctx context.Context) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, "SHOW CATALOGS")
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}
	defer rows.Close()

	var catalogs []string
	for rows.Next() {
		var catalog string
		if err := rows.Scan(&catalog); err != nil {
			return nil, fmt.Errorf("failed to scan catalog: %w", err)
		}
		catalogs = append(catalogs, catalog)
	}
	return catalogs, nil
}

// ListSchemas returns schemas in the given catalog.
func (c *Client) ListSchemas(ctx context.Context, catalog string) ([]string, error) {
	if catalog == "" {
		catalog = c.config.Catalog
	}

	query := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog) // #nosec G201 -- catalog is from config, not user input
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, fmt.Errorf("failed to scan schema: %w", err)
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}

// ListTables returns tables in the given catalog and schema.
func (c *Client) ListTables(ctx context.Context, catalog, schema string) ([]TableInfo, error) {
	if catalog == "" {
		catalog = c.config.Catalog
	}
	if schema == "" {
		schema = c.config.Schema
	}

	query := fmt.Sprintf("SHOW TABLES FROM %s.%s", catalog, schema) // #nosec G201 -- catalog/schema are from config, not user input
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}
		tables = append(tables, TableInfo{
			Catalog: catalog,
			Schema:  schema,
			Name:    name,
			Type:    "TABLE",
		})
	}
	return tables, nil
}

// DescribeTable returns detailed information about a table.
func (c *Client) DescribeTable(ctx context.Context, catalog, schema, table string) (*TableInfo, error) {
	if catalog == "" {
		catalog = c.config.Catalog
	}
	if schema == "" {
		schema = c.config.Schema
	}

	query := fmt.Sprintf("DESCRIBE %s.%s.%s", catalog, schema, table) // #nosec G201 -- catalog/schema/table from config or validated input
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	info := &TableInfo{
		Catalog: catalog,
		Schema:  schema,
		Name:    table,
		Type:    "TABLE",
		Columns: make([]ColumnDef, 0),
	}

	for rows.Next() {
		var col ColumnDef
		var extra, comment sql.NullString
		if err := rows.Scan(&col.Name, &col.Type, &extra, &comment); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		if extra.Valid {
			col.Nullable = extra.String
		}
		if comment.Valid {
			col.Comment = comment.String
		}
		info.Columns = append(info.Columns, col)
	}

	return info, nil
}

// convertValue converts database values to JSON-friendly types.
func convertValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		// Try to parse as JSON
		var jsonVal any
		if err := json.Unmarshal(val, &jsonVal); err == nil {
			return jsonVal
		}
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return val
	}
}
