package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/trinodb/trino-go-client/trino"
)

func newStreamClient(t *testing.T) (*Client, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewWithDB(db, Config{Host: "localhost", Port: 8080, User: "test", Timeout: 30 * time.Second}), mock
}

func threeRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)).AddRow(int64(3))
}

// drain reads every row the cursor yields and returns their ids.
func drain(t *testing.T, cur *RowCursor) []int64 {
	t.Helper()
	var ids []int64
	for cur.Next() {
		id, ok := cur.Row()["id"].(int64)
		if !ok {
			t.Fatalf("id: got %#v, want int64", cur.Row()["id"])
		}
		ids = append(ids, id)
	}
	return ids
}

func TestQueryStream_Limit(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		wantIDs       int
		wantTruncated bool
	}{
		{name: "zero limit reads every row", limit: 0, wantIDs: 3},
		{name: "negative limit reads every row", limit: -1, wantIDs: 3},
		{name: "limit below the result truncates", limit: 2, wantIDs: 2, wantTruncated: true},
		{name: "limit equal to the result does not truncate", limit: 3, wantIDs: 3},
		{name: "limit above the result does not truncate", limit: 10, wantIDs: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newStreamClient(t)
			mock.ExpectQuery("SELECT").WillReturnRows(threeRows())

			cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{Limit: tt.limit})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = cur.Close() }()

			ids := drain(t, cur)
			if len(ids) != tt.wantIDs {
				t.Errorf("got %d rows %v, want %d", len(ids), ids, tt.wantIDs)
			}
			for i, id := range ids {
				if id != int64(i+1) {
					t.Errorf("row %d: got id %d, want %d", i, id, i+1)
				}
			}
			if err := cur.Err(); err != nil {
				t.Errorf("unexpected iteration error: %v", err)
			}
			stats := cur.Stats()
			if stats.LimitApplied != max(tt.limit, 0) {
				t.Errorf("LimitApplied: got %d, want %d", stats.LimitApplied, max(tt.limit, 0))
			}
			if stats.RowCount != tt.wantIDs || stats.Truncated != tt.wantTruncated {
				t.Errorf("stats: got rows %d truncated %v, want %d %v",
					stats.RowCount, stats.Truncated, tt.wantIDs, tt.wantTruncated)
			}
		})
	}
}

func TestQueryStream_ColumnsBeforeNextAndRowLifetime(t *testing.T) {
	client, mock := newStreamClient(t)
	mock.ExpectQuery("SELECT").WillReturnRows(threeRows())

	cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = cur.Close() }()

	if cols := cur.Columns(); len(cols) != 1 || cols[0].Name != "id" {
		t.Fatalf("columns before Next: got %+v", cols)
	}
	if cur.Row() != nil {
		t.Errorf("Row before Next: got %v, want nil", cur.Row())
	}

	if !cur.Next() {
		t.Fatal("expected a first row")
	}
	first := cur.Row()
	if !cur.Next() {
		t.Fatal("expected a second row")
	}
	if first["id"] != int64(1) || cur.Row()["id"] != int64(2) {
		t.Errorf("a kept row changed when the cursor advanced: first %v, current %v", first, cur.Row())
	}

	drain(t, cur)
	if cur.Row() != nil {
		t.Errorf("Row after the end: got %v, want nil", cur.Row())
	}
}

func TestQueryStream_RawValues(t *testing.T) {
	ts := time.Date(2024, 5, 1, 12, 34, 56, 789123000, time.UTC)
	for _, raw := range []bool{false, true} {
		client, mock := newStreamClient(t)
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"at"}).AddRow(ts))

		cur, err := client.QueryStream(context.Background(), "SELECT at", QueryOptions{RawValues: raw})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cur.Next() {
			t.Fatalf("raw=%v: expected a row, err %v", raw, cur.Err())
		}
		got := cur.Row()["at"]
		if raw {
			if v, ok := got.(time.Time); !ok || !v.Equal(ts) {
				t.Errorf("raw: got %#v, want time.Time %v", got, ts)
			}
		} else if got != "2024-05-01T12:34:56.789123Z" {
			t.Errorf("rendered: got %#v", got)
		}
		_ = cur.Close()
	}
}

func TestQueryStream_Errors(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("query error", func(t *testing.T) {
		client, mock := newStreamClient(t)
		mock.ExpectQuery("SELECT").WillReturnError(errBoom)

		cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{})
		if cur != nil || !errors.Is(err, errBoom) {
			t.Fatalf("got cursor %v err %v, want nil and %v", cur, err, errBoom)
		}
	})

	t.Run("iteration error mid-stream", func(t *testing.T) {
		client, mock := newStreamClient(t)
		mock.ExpectQuery("SELECT").WillReturnRows(threeRows().RowError(1, errBoom))

		cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = cur.Close() }()

		if ids := drain(t, cur); len(ids) != 1 {
			t.Errorf("got rows %v, want only the row before the error", ids)
		}
		if !errors.Is(cur.Err(), errBoom) {
			t.Errorf("Err: got %v, want %v", cur.Err(), errBoom)
		}
		if cur.Next() {
			t.Error("Next after an error returned true")
		}
	})

	t.Run("iteration error on the row past the limit", func(t *testing.T) {
		client, mock := newStreamClient(t)
		mock.ExpectQuery("SELECT").WillReturnRows(threeRows().RowError(2, errBoom))

		cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = cur.Close() }()

		drain(t, cur)
		if !errors.Is(cur.Err(), errBoom) {
			t.Errorf("Err: got %v, want %v", cur.Err(), errBoom)
		}
		if cur.Stats().Truncated {
			t.Error("a failed read past the limit was reported as truncation")
		}
	})

	t.Run("scan error", func(t *testing.T) {
		client, mock := newStreamClient(t)
		rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1))
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = cur.Close() }()

		cur.valuePtrs = append(cur.valuePtrs, new(any)) // more destinations than columns
		if cur.Next() {
			t.Fatal("Next succeeded with a failing scan")
		}
		if cur.Err() == nil || cur.Row() != nil {
			t.Errorf("got err %v row %v, want a scan error and no row", cur.Err(), cur.Row())
		}
	})
}

func TestQueryStream_Close(t *testing.T) {
	client, mock := newStreamClient(t)
	errClose := errors.New("close failed")
	mock.ExpectQuery("SELECT").WillReturnRows(threeRows().CloseError(errClose))

	cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cur.Next() {
		t.Fatal("expected a row before Close")
	}

	if err := cur.Close(); !errors.Is(err, errClose) {
		t.Errorf("first Close: got %v, want %v", err, errClose)
	}
	if err := cur.Close(); err != nil {
		t.Errorf("second Close: got %v, want nil", err)
	}
	if cur.Next() {
		t.Error("Next after Close returned true")
	}
	if cur.Stats().RowCount != 1 {
		t.Errorf("Stats after Close: got %d rows, want 1", cur.Stats().RowCount)
	}
	if cur.Row() != nil {
		t.Errorf("Row after Close: got %v, want nil", cur.Row())
	}
}

func TestRowCursor_StatsQueryID(t *testing.T) {
	client, mock := newStreamClient(t)
	mock.ExpectQuery("SELECT").WillReturnRows(threeRows())

	cur, err := client.QueryStream(context.Background(), "SELECT id", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = cur.Close() }()

	if got := cur.Stats().QueryID; got != "" {
		t.Errorf("QueryID without a progress updater: got %q, want empty", got)
	}

	// sqlmock rejects the progress callback, so stand in for the one the Trino
	// driver would have been given.
	cur.progress = &queryProgressUpdater{}
	cur.progress.Update(trino.QueryProgressInfo{QueryId: "20240115_123456_00001_abcde"})
	if got := cur.Stats().QueryID; got != "20240115_123456_00001_abcde" {
		t.Errorf("QueryID: got %q, want the updater's", got)
	}
}
