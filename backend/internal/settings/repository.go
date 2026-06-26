package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrNotStored signals that no settings row exists yet.
var ErrNotStored = errors.New("settings not stored")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Get reads the stored settings document, or ErrNotStored if none exists yet.
func (r *Repository) Get(ctx context.Context) (Settings, error) {
	var data string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM settings WHERE id = 1`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, ErrNotStored
	}
	if err != nil {
		return Settings{}, fmt.Errorf("query settings: %w", err)
	}

	var s Settings
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	return s, nil
}

// Save upserts the single settings row.
func (r *Repository) Save(ctx context.Context, s Settings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO settings (id, data, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		string(data), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
