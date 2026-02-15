package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a sql.DB connection to a SQLite database.
type DB struct {
	db *sql.DB
}

// New opens a SQLite database at the given DSN and runs migrations.
// Use ":memory:" for an in-memory database.
func New(dsn string) (*DB, error) {
	connStr := dsn
	if dsn != ":memory:" {
		connStr = dsn + "?_journal_mode=WAL&_foreign_keys=on"
	} else {
		connStr = ":memory:?_foreign_keys=on"
	}

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

func (s *DB) migrate() error {
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}
	if _, err := s.db.Exec(ftsSchema); err != nil {
		return fmt.Errorf("failed to apply FTS schema: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *DB) Close() error {
	return s.db.Close()
}
