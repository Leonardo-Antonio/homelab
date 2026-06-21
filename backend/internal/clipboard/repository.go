package clipboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("clipboard item not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Item, int, error) {
	total, err := r.count(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, text, created_at
		FROM clipboard_items
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query clipboard items: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0, limit)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, 0, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate clipboard items: %w", err)
	}

	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id string) (Item, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, text, created_at
		FROM clipboard_items
		WHERE id = ?`, id)

	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}

	return item, nil
}

func (r *Repository) Create(ctx context.Context, item Item) (Item, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO clipboard_items (id, text, created_at)
		VALUES (?, ?, ?)`, item.ID, item.Text, item.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Item{}, fmt.Errorf("insert clipboard item: %w", err)
	}

	return item, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM clipboard_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete clipboard item: %w", err)
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

func (r *Repository) DeleteAll(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM clipboard_items`); err != nil {
		return fmt.Errorf("delete clipboard items: %w", err)
	}

	return nil
}

func (r *Repository) count(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clipboard_items`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count clipboard items: %w", err)
	}

	return total, nil
}

type itemScanner interface {
	Scan(dest ...any) error
}

func scanItem(scanner itemScanner) (Item, error) {
	var item Item
	var createdAt string

	if err := scanner.Scan(&item.ID, &item.Text, &createdAt); err != nil {
		return Item{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Item{}, fmt.Errorf("parse clipboard item created_at: %w", err)
	}

	item.CreatedAt = parsedCreatedAt.UTC()
	return item, nil
}
