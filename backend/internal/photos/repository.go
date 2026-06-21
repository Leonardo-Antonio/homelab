package photos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("photo not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Photo, int, error) {
	total, err := r.count(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, file_name, content_type, size_bytes, width, height, created_at
		FROM photos
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query photos: %w", err)
	}
	defer rows.Close()

	items := make([]Photo, 0, limit)
	for rows.Next() {
		item, err := scanPhoto(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate photos: %w", err)
	}

	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id string) (Photo, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, file_name, content_type, size_bytes, width, height, created_at
		FROM photos
		WHERE id = ?`, id)

	item, err := scanPhoto(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Photo{}, ErrNotFound
	}
	if err != nil {
		return Photo{}, err
	}

	return item, nil
}

func (r *Repository) Create(ctx context.Context, photo Photo) (Photo, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO photos (id, file_name, content_type, size_bytes, width, height, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		photo.ID,
		photo.FileName,
		photo.ContentType,
		photo.SizeBytes,
		photo.Width,
		photo.Height,
		photo.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return Photo{}, fmt.Errorf("insert photo: %w", err)
	}

	return photo, nil
}

func (r *Repository) Delete(ctx context.Context, id string) (Photo, error) {
	photo, err := r.Get(ctx, id)
	if err != nil {
		return Photo{}, err
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM photos WHERE id = ?`, id)
	if err != nil {
		return Photo{}, fmt.Errorf("delete photo: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Photo{}, fmt.Errorf("read deleted row count: %w", err)
	}
	if affected == 0 {
		return Photo{}, ErrNotFound
	}

	return photo, nil
}

func (r *Repository) count(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM photos`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count photos: %w", err)
	}

	return total, nil
}

type photoScanner interface {
	Scan(dest ...any) error
}

func scanPhoto(scanner photoScanner) (Photo, error) {
	var photo Photo
	var createdAt string
	if err := scanner.Scan(
		&photo.ID,
		&photo.FileName,
		&photo.ContentType,
		&photo.SizeBytes,
		&photo.Width,
		&photo.Height,
		&createdAt,
	); err != nil {
		return Photo{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Photo{}, fmt.Errorf("parse photo created_at: %w", err)
	}

	photo.CreatedAt = parsedCreatedAt.UTC()
	photo.URL = "/api/v1/photos/" + photo.ID + "/file"
	return photo, nil
}
