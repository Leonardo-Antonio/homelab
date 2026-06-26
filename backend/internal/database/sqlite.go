package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS clipboard_items (
			id TEXT PRIMARY KEY,
			text TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_clipboard_items_created_at
			ON clipboard_items(created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS photos (
			id TEXT PRIMARY KEY,
			file_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_created_at
			ON photos(created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS notes (
			id         TEXT PRIMARY KEY,
			parent_id  TEXT REFERENCES notes(id) ON DELETE CASCADE,
			type       TEXT NOT NULL CHECK(type IN ('dir', 'note')),
			name       TEXT NOT NULL,
			content    TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_notes_parent_id ON notes(parent_id);`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("run migration statement: %w", err)
		}
	}

	return nil
}
