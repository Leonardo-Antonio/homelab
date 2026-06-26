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
		// ── Storage (Drive-like) ────────────────────────────────────────────
		// Content-addressed blobs: one row per unique SHA-256. ref_count is
		// maintained transactionally by triggers so a physical blob is only
		// ever garbage-collected once no node references it.
		`CREATE TABLE IF NOT EXISTS storage_blobs (
			id         TEXT PRIMARY KEY,      -- sha-256 hex of the content
			size_bytes INTEGER NOT NULL,
			ref_count  INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		// Hierarchical tree of folders and files. A file points at exactly one
		// blob; folders have a NULL blob_id. UNIQUE(parent_id, name) prevents
		// duplicate names within the same folder.
		`CREATE TABLE IF NOT EXISTS storage_nodes (
			id           TEXT PRIMARY KEY,
			parent_id    TEXT REFERENCES storage_nodes(id) ON DELETE CASCADE,
			type         TEXT NOT NULL CHECK(type IN ('dir', 'file')),
			name         TEXT NOT NULL,
			blob_id      TEXT REFERENCES storage_blobs(id),
			content_type TEXT,
			size_bytes   INTEGER NOT NULL DEFAULT 0,
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_storage_nodes_parent_id ON storage_nodes(parent_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_storage_nodes_unique_name
			ON storage_nodes(IFNULL(parent_id, ''), name);`,
		// Keep blob ref counts in lockstep with node rows, even for cascading
		// deletes the application never sees row-by-row.
		`CREATE TRIGGER IF NOT EXISTS storage_blob_ref_inc
			AFTER INSERT ON storage_nodes
			WHEN NEW.blob_id IS NOT NULL
			BEGIN
				UPDATE storage_blobs SET ref_count = ref_count + 1 WHERE id = NEW.blob_id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS storage_blob_ref_dec
			AFTER DELETE ON storage_nodes
			WHEN OLD.blob_id IS NOT NULL
			BEGIN
				UPDATE storage_blobs SET ref_count = ref_count - 1 WHERE id = OLD.blob_id;
			END;`,
		`CREATE TRIGGER IF NOT EXISTS storage_blob_ref_move
			AFTER UPDATE OF blob_id ON storage_nodes
			WHEN IFNULL(OLD.blob_id, '') <> IFNULL(NEW.blob_id, '')
			BEGIN
				UPDATE storage_blobs SET ref_count = ref_count - 1 WHERE id = OLD.blob_id;
				UPDATE storage_blobs SET ref_count = ref_count + 1 WHERE id = NEW.blob_id;
			END;`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("run migration statement: %w", err)
		}
	}

	return nil
}
