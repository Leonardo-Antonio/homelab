package notes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]Node, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parent_id, type, name, created_at, updated_at
		FROM notes
		ORDER BY type DESC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (NoteDetail, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parent_id, type, name, created_at, updated_at, COALESCE(content, '')
		FROM notes
		WHERE id = ?`, id)

	var detail NoteDetail
	var parentID sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&detail.ID, &parentID, &detail.Type, &detail.Name,
		&createdAt, &updatedAt, &detail.Content,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return NoteDetail{}, ErrNotFound
	}
	if err != nil {
		return NoteDetail{}, fmt.Errorf("scan note: %w", err)
	}

	if parentID.Valid {
		detail.ParentID = &parentID.String
	}

	if detail.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return NoteDetail{}, fmt.Errorf("parse note created_at: %w", err)
	}
	if detail.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return NoteDetail{}, fmt.Errorf("parse note updated_at: %w", err)
	}

	return detail, nil
}

func (r *Repository) Create(ctx context.Context, node Node, content string) (Node, error) {
	now := node.CreatedAt.UTC().Format(time.RFC3339Nano)
	var nullableContent sql.NullString
	if node.Type == TypeNote {
		nullableContent = sql.NullString{String: content, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notes (id, parent_id, type, name, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		node.ID, nullableParentID(node.ParentID), node.Type, node.Name,
		nullableContent, now, now)
	if err != nil {
		return Node{}, fmt.Errorf("insert note: %w", err)
	}

	return node, nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest, now time.Time) (Node, error) {
	nowStr := now.UTC().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `
		UPDATE notes
		SET name      = ?,
		    content   = CASE WHEN type = 'note' THEN ? ELSE content END,
		    parent_id = ?,
		    updated_at = ?
		WHERE id = ?`,
		req.Name, req.Content, nullableParentID(req.ParentID), nowStr, id)
	if err != nil {
		return Node{}, fmt.Errorf("update note: %w", err)
	}

	return r.getNode(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted row count: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) getNode(ctx context.Context, id string) (Node, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parent_id, type, name, created_at, updated_at
		FROM notes WHERE id = ?`, id)
	return scanNode(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(s rowScanner) (Node, error) {
	var node Node
	var parentID sql.NullString
	var createdAt, updatedAt string

	if err := s.Scan(&node.ID, &parentID, &node.Type, &node.Name, &createdAt, &updatedAt); err != nil {
		return Node{}, fmt.Errorf("scan node: %w", err)
	}

	if parentID.Valid {
		node.ParentID = &parentID.String
	}

	var err error
	if node.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Node{}, fmt.Errorf("parse node created_at: %w", err)
	}
	if node.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return Node{}, fmt.Errorf("parse node updated_at: %w", err)
	}

	return node, nil
}

func nullableParentID(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
