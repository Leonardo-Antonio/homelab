package clipboard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var ErrInvalidText = errors.New("text is required and must not exceed the max length")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, page, pageSize int) (ListItemsResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	offset := (page - 1) * pageSize
	limit := pageSize
	items, total, err := s.repository.List(ctx, limit, offset)
	if err != nil {
		return ListItemsResponse{}, err
	}

	pages := 0
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
	}

	return ListItemsResponse{
		Items:       items,
		Page:        page,
		PageSize:    pageSize,
		Pages:       pages,
		Total:       total,
		HasNext:     page < pages,
		HasPrevious: page > 1 && pages > 0,
	}, nil
}

func (s *Service) Get(ctx context.Context, id string) (Item, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, text string) (Item, error) {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > MaxTextLength {
		return Item{}, ErrInvalidText
	}

	item := Item{
		ID:        newID(),
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}

	return s.repository.Create(ctx, item)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}

func (s *Service) DeleteAll(ctx context.Context) error {
	return s.repository.DeleteAll(ctx)
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(bytes)
}
