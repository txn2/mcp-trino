package client

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/trinodb/trino-go-client/trino"
)

// Session headers the Trino driver sends for a named query argument; a
// per-query value replaces the connection's.
const (
	trinoCatalogHeader = "X-Trino-Catalog"
	trinoSchemaHeader  = "X-Trino-Schema"
)

// RowCursor yields a query's rows one at a time, holding only the current
// row in memory. It is not safe for concurrent use. The caller must Close it.
type RowCursor struct {
	rows     *sql.Rows
	cancel   context.CancelFunc
	columns  []ColumnInfo
	raw      bool
	limit    int
	start    time.Time
	progress *queryProgressUpdater
	tracker  *queryTracker

	values    []any
	valuePtrs []any
	row       map[string]any
	rowCount  int
	truncated bool
	err       error
	closed    bool
}

// QueryStream executes sqlQuery and returns a cursor over its rows, so a
// result is read incrementally rather than buffered whole. The cursor honors
// opts.RawValues and opts.Timeout; the timeout bounds the whole stream, not
// each row. The session catalog and schema follow opts.Catalog and
// opts.Schema for this query only, as QueryOptions describes. When positive,
// opts.Limit caps the rows read; unlike Query, zero or negative means no cap.
// Canceling ctx ends iteration, and Err then reports the cancellation.
func (c *Client) QueryStream(ctx context.Context, sqlQuery string, opts QueryOptions) (*RowCursor, error) {
	start := time.Now()

	session, err := c.sessionArgs(opts)
	if err != nil {
		return nil, err
	}

	timeout := c.config.Timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	tracker := &queryTracker{}
	ctx = withQueryTracker(ctx, tracker)

	rows, progress, err := c.openRows(ctx, sqlQuery, session)
	if err != nil {
		cancel()
		return nil, newQueryError(err, tracker, progress, start)
	}

	columns, err := resultColumns(rows)
	if err != nil {
		_ = rows.Close()
		cancel()
		return nil, err
	}

	cur := &RowCursor{
		rows:      rows,
		cancel:    cancel,
		columns:   columns,
		raw:       opts.RawValues,
		limit:     max(opts.Limit, 0),
		start:     start,
		progress:  progress,
		tracker:   tracker,
		values:    make([]any, len(columns)),
		valuePtrs: make([]any, len(columns)),
	}
	for i := range cur.values {
		cur.valuePtrs[i] = &cur.values[i]
	}
	return cur, nil
}

// sessionArgs returns the per-query session headers for opts' catalog and
// schema overrides; the connection already sends the configured ones. A
// catalog override also replaces the schema, cleared when opts.Schema is
// empty, because the configured schema belongs to the configured catalog.
func (c *Client) sessionArgs(opts QueryOptions) ([]any, error) {
	if opts.Catalog != "" {
		return []any{
			sql.Named(trinoCatalogHeader, opts.Catalog),
			sql.Named(trinoSchemaHeader, opts.Schema),
		}, nil
	}
	if opts.Schema == "" {
		return nil, nil
	}
	if c.config.Catalog == "" {
		return nil, fmt.Errorf("schema %q needs a catalog: set QueryOptions.Catalog or Config.Catalog", opts.Schema)
	}
	return []any{sql.Named(trinoSchemaHeader, opts.Schema)}, nil
}

// openRows runs the query with session and a progress callback that captures
// the Trino query ID. A driver that rejects the callback argument (sqlmock, in
// tests) gets the query with session alone, and the returned updater is nil.
// The updater is returned with an error too, holding any ID it captured.
func (c *Client) openRows(ctx context.Context, sqlQuery string, session []any) (*sql.Rows, *queryProgressUpdater, error) {
	progress := &queryProgressUpdater{}
	args := append([]any{
		sql.Named("X-Trino-Progress-Callback", trino.ProgressUpdater(progress)),
		sql.Named("X-Trino-Progress-Callback-Period", 100*time.Millisecond),
	}, session...)
	rows, err := c.db.QueryContext(ctx, sqlQuery, args...)
	if err == nil {
		return rows, progress, nil
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		return nil, progress, fmt.Errorf("query failed: %w", err)
	}
	rows, err = c.db.QueryContext(ctx, sqlQuery, session...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil, nil
}

// resultColumns describes each column of rows.
func resultColumns(rows *sql.Rows) ([]ColumnInfo, error) {
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
		if precision, scale, ok := ct.DecimalSize(); ok {
			columns[i].Precision, columns[i].Scale = precision, scale
		}
	}
	return columns, nil
}

// Columns describes the result's columns. It is available before the first
// call to Next.
func (cur *RowCursor) Columns() []ColumnInfo {
	return cur.columns
}

// Next advances to the next row. It returns false at the end of the result,
// when the limit is reached, or on an error, which Err then reports.
func (cur *RowCursor) Next() bool {
	if cur.closed || cur.err != nil {
		return false
	}
	if cur.limit > 0 && cur.rowCount >= cur.limit {
		// One row past the limit says whether the limit cut the result.
		cur.truncated = cur.advance()
		cur.row = nil
		return false
	}
	if !cur.advance() {
		cur.row = nil
		return false
	}
	if err := cur.rows.Scan(cur.valuePtrs...); err != nil {
		cur.err = fmt.Errorf("failed to scan row: %w", err)
		cur.row = nil
		return false
	}

	row := make(map[string]any, len(cur.columns))
	for i, col := range cur.columns {
		row[col.Name] = resultValue(cur.values[i], cur.raw)
	}
	cur.row = row
	cur.rowCount++
	return true
}

// advance moves the driver to its next row, recording an iteration error when
// that is why there is none. The driver's rows are closed by the time Next
// returns false, so a cancel the driver sent has been answered.
func (cur *RowCursor) advance() bool {
	if cur.rows.Next() {
		return true
	}
	if err := cur.rows.Err(); err != nil {
		cur.err = newQueryError(fmt.Errorf("row iteration error: %w", err), cur.tracker, cur.progress, cur.start)
	}
	return false
}

// Row returns the current row, or nil before the first Next, after Next
// returns false, or after Close. Each call to Next makes a new map, so the
// caller may keep it.
func (cur *RowCursor) Row() map[string]any {
	return cur.row
}

// Err returns the error that ended iteration, if any.
func (cur *RowCursor) Err() error {
	return cur.err
}

// Stats reports the rows read so far, the elapsed time, whether the limit cut
// the result, and the Trino query ID when the driver reported one.
func (cur *RowCursor) Stats() QueryStats {
	stats := QueryStats{
		RowCount:     cur.rowCount,
		DurationMs:   time.Since(cur.start).Milliseconds(),
		Truncated:    cur.truncated,
		LimitApplied: cur.limit,
	}
	if cur.progress != nil {
		stats.QueryID = cur.progress.QueryID()
	}
	return stats
}

// Close releases the result and cancels the query's context. It is safe to
// call more than once.
func (cur *RowCursor) Close() error {
	if cur.closed {
		return nil
	}
	cur.closed = true
	cur.row = nil
	err := cur.rows.Close()
	cur.cancel()
	return err
}
