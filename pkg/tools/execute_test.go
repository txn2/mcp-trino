package tools

import "testing"

func TestIsWriteSQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		isWrite bool
	}{
		// Write operations
		{"INSERT", "INSERT INTO users VALUES (1, 'Alice')", true},
		{"UPDATE", "UPDATE users SET name = 'Bob'", true},
		{"DELETE", "DELETE FROM users WHERE id = 1", true},
		{"DROP", "DROP TABLE users", true},
		{"CREATE", "CREATE TABLE new_table (id INT)", true},
		{"ALTER", "ALTER TABLE users ADD COLUMN email VARCHAR", true},
		{"TRUNCATE", "TRUNCATE TABLE users", true},
		{"GRANT", "GRANT SELECT ON users TO analyst", true},
		{"REVOKE", "REVOKE SELECT ON users FROM analyst", true},
		{"MERGE", "MERGE INTO target USING source ON t.id = s.id", true},
		{"CALL", "CALL system.sync_partition_metadata('catalog', 'schema', 'table')", true},
		{"EXECUTE", "EXECUTE my_prepared_statement", true},

		// Case insensitive
		{"insert lowercase", "insert into users values (1)", true},
		{"Insert mixed", "Insert Into users VALUES (1)", true},

		// Leading whitespace
		{"leading spaces", "   INSERT INTO users VALUES (1)", true},
		{"leading tab", "\tDROP TABLE users", true},

		// SQL comments before keyword
		{"line comment", "-- comment\nINSERT INTO users VALUES (1)", true},
		{"block comment", "/* comment */INSERT INTO users VALUES (1)", true},

		// Read operations
		{"SELECT", "SELECT * FROM users", false},
		{"select lowercase", "select id from users", false},
		{"SHOW", "SHOW TABLES", false},
		{"DESCRIBE", "DESCRIBE users", false},
		{"EXPLAIN", "EXPLAIN SELECT * FROM users", false},
		{"WITH CTE", "WITH cte AS (SELECT 1) SELECT * FROM cte", false},

		// Edge cases
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"SELECT with INSERT in value", "SELECT * FROM users WHERE name = 'INSERT'", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWriteSQL(tt.sql)
			if got != tt.isWrite {
				t.Errorf("IsWriteSQL(%q) = %v, want %v", tt.sql, got, tt.isWrite)
			}
		})
	}
}
