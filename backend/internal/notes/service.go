package notes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Node, error) {
	nodes, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		return []Node{}, nil
	}
	return nodes, nil
}

func (s *Service) Get(ctx context.Context, id string) (NoteDetail, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Node, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > MaxNameLength {
		return Node{}, ErrInvalidName
	}
	if req.Type != TypeDir && req.Type != TypeNote {
		return Node{}, ErrInvalidType
	}
	if len(req.Content) > MaxContentLength {
		return Node{}, ErrContentTooLong
	}

	now := time.Now().UTC()
	node := Node{
		ID:        newID(),
		ParentID:  req.ParentID,
		Type:      req.Type,
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return s.repo.Create(ctx, node, req.Content)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Node, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return Node{}, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > MaxNameLength {
		return Node{}, ErrInvalidName
	}
	if len(req.Content) > MaxContentLength {
		return Node{}, ErrContentTooLong
	}

	return s.repo.Update(ctx, id, req, time.Now().UTC())
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}
