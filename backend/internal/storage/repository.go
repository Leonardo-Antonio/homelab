package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Reads ───────────────────────────────────────────────────────────────────

// ListChildren returns the direct children of parentID (nil = root), folders
// first then files, each alphabetically.
func (r *Repository) ListChildren(ctx context.Context, parentID *string) ([]Node, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parent_id, type, name, blob_id, content_type, size_bytes, created_at, updated_at
		FROM storage_nodes
		WHERE IFNULL(parent_id, '') = IFNULL(?, '')
		ORDER BY type DESC, name COLLATE NOCASE ASC`,
		nullable(parentID))
	if err != nil {
		return nil, fmt.Errorf("query children: %w", err)
	}
	defer rows.Close()

	nodes := make([]Node, 0)
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (Node, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parent_id, type, name, blob_id, content_type, size_bytes, created_at, updated_at
		FROM storage_nodes
		WHERE id = ?`, id)

	node, err := scanNode(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNotFound
	}
	return node, err
}

// Ancestors walks parent links from the node up to the root, returning the
// path ordered root → ... → node.
func (r *Repository) Ancestors(ctx context.Context, id string) ([]Breadcrumb, error) {
	var trail []Breadcrumb
	current := &id
	// Bound the walk to avoid an infinite loop should data ever be corrupted.
	for depth := 0; current != nil && depth < 4096; depth++ {
		row := r.db.QueryRowContext(ctx, `SELECT id, parent_id, name FROM storage_nodes WHERE id = ?`, *current)
		var bc Breadcrumb
		var parentID sql.NullString
		if err := row.Scan(&bc.ID, &parentID, &bc.Name); errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		} else if err != nil {
			return nil, fmt.Errorf("scan ancestor: %w", err)
		}
		trail = append([]Breadcrumb{bc}, trail...)
		if parentID.Valid {
			current = &parentID.String
		} else {
			current = nil
		}
	}
	return trail, nil
}

// ── Writes ──────────────────────────────────────────────────────────────────

// validateParent ensures parentID (when set) refers to an existing folder.
func (r *Repository) validateParent(ctx context.Context, q querier, parentID *string) error {
	if parentID == nil {
		return nil
	}
	var nodeType string
	err := q.QueryRowContext(ctx, `SELECT type FROM storage_nodes WHERE id = ?`, *parentID).Scan(&nodeType)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && nodeType != TypeDir) {
		return ErrInvalidParent
	}
	return err
}

func (r *Repository) CreateFolder(ctx context.Context, node Node) (Node, error) {
	if err := r.validateParent(ctx, r.db, node.ParentID); err != nil {
		return Node{}, err
	}
	ts := node.CreatedAt.UTC().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO storage_nodes (id, parent_id, type, name, size_bytes, created_at, updated_at)
		VALUES (?, ?, 'dir', ?, 0, ?, ?)`,
		node.ID, nullable(node.ParentID), node.Name, ts, ts)
	if err != nil {
		return Node{}, mapWriteErr(err)
	}
	return node, nil
}

// CreateFile inserts a file node and upserts its backing blob inside a single
// transaction. The blob row must exist before the node so the ref-count
// trigger has a row to increment; the physical blob is written to disk by the
// service *before* this call, guaranteeing durable-content-before-metadata.
func (r *Repository) CreateFile(ctx context.Context, node Node, blobSize int64) (Node, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Node{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.validateParent(ctx, tx, node.ParentID); err != nil {
		return Node{}, err
	}

	ts := node.CreatedAt.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO storage_blobs (id, size_bytes, ref_count, created_at)
		VALUES (?, ?, 0, ?)
		ON CONFLICT(id) DO NOTHING`,
		node.blobID, blobSize, ts); err != nil {
		return Node{}, fmt.Errorf("upsert blob: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO storage_nodes
			(id, parent_id, type, name, blob_id, content_type, size_bytes, created_at, updated_at)
		VALUES (?, ?, 'file', ?, ?, ?, ?, ?, ?)`,
		node.ID, nullable(node.ParentID), node.Name, node.blobID,
		node.ContentType, node.SizeBytes, ts, ts); err != nil {
		return Node{}, mapWriteErr(err)
	}

	if err := tx.Commit(); err != nil {
		return Node{}, fmt.Errorf("commit tx: %w", err)
	}
	return node, nil
}

// Update applies a validated rename/move to a node. parentID semantics are
// resolved by the service; here both values are final.
func (r *Repository) Update(ctx context.Context, id, name string, parentID *string, now time.Time) (Node, error) {
	if err := r.validateParent(ctx, r.db, parentID); err != nil {
		return Node{}, err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE storage_nodes
		SET name = ?, parent_id = ?, updated_at = ?
		WHERE id = ?`,
		name, nullable(parentID), now.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return Node{}, mapWriteErr(err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return Node{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

// Delete removes a node (and, via ON DELETE CASCADE, its whole subtree), then
// returns the digests of blobs that became unreferenced so the caller can
// garbage-collect the physical files. Triggers keep ref_count exact.
func (r *Repository) Delete(ctx context.Context, id string) ([]string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `DELETE FROM storage_nodes WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("delete node: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, ErrNotFound
	}

	orphans, err := collectOrphans(ctx, tx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM storage_blobs WHERE ref_count <= 0`); err != nil {
		return nil, fmt.Errorf("delete orphan blob rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return orphans, nil
}

// SweepOrphans deletes any blob rows that ended up unreferenced (e.g. after a
// crash mid-delete) and returns their digests for physical cleanup. Safe to
// run at startup.
func (r *Repository) SweepOrphans(ctx context.Context) ([]string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	orphans, err := collectOrphans(ctx, tx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM storage_blobs WHERE ref_count <= 0`); err != nil {
		return nil, fmt.Errorf("delete orphan blob rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return orphans, nil
}

func collectOrphans(ctx context.Context, q querier) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT id FROM storage_blobs WHERE ref_count <= 0`)
	if err != nil {
		return nil, fmt.Errorf("query orphan blobs: %w", err)
	}
	defer rows.Close()

	var digests []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan orphan blob: %w", err)
		}
		digests = append(digests, id)
	}
	return digests, rows.Err()
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// querier is satisfied by both *sql.DB and *sql.Tx.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(s rowScanner) (Node, error) {
	var node Node
	var parentID, blobID, contentType sql.NullString
	var createdAt, updatedAt string

	if err := s.Scan(
		&node.ID, &parentID, &node.Type, &node.Name,
		&blobID, &contentType, &node.SizeBytes, &createdAt, &updatedAt,
	); err != nil {
		return Node{}, err
	}

	if parentID.Valid {
		node.ParentID = &parentID.String
	}
	node.blobID = blobID.String
	node.ContentType = contentType.String

	var err error
	if node.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Node{}, fmt.Errorf("parse created_at: %w", err)
	}
	if node.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return Node{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return node, nil
}

func nullable(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// mapWriteErr translates SQLite constraint failures into domain errors.
func mapWriteErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") {
		return ErrNameConflict
	}
	return fmt.Errorf("storage write: %w", err)
}
