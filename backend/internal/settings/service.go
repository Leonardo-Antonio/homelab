package settings

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the persisted settings, falling back to (and never failing on)
// defaults when nothing has been stored yet. The result is always normalized.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	stored, err := s.repo.Get(ctx)
	if errors.Is(err, ErrNotStored) {
		return Default(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	return stored.normalize(), nil
}

// Update validates, normalizes and persists a new settings document, returning
// the stored result.
func (s *Service) Update(ctx context.Context, incoming Settings) (Settings, error) {
	normalized := incoming.normalize()
	if err := normalized.validate(); err != nil {
		return Settings{}, err
	}
	if err := s.repo.Save(ctx, normalized); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}
