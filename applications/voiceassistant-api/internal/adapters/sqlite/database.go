package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Register the pure Go SQLite driver with database/sql.
	_ "modernc.org/sqlite"
)

func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`CREATE TABLE IF NOT EXISTS audiomessages (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL UNIQUE,
			transcription TEXT NOT NULL,
			analysis TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initialize database: %w", err)
		}
	}
	return db, nil
}
