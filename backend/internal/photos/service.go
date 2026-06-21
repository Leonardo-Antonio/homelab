package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"time"
)

var ErrInvalidPhoto = errors.New("photo must be a valid jpeg or png image")

type Service struct {
	repository *Repository
	storageDir string
}

func NewService(repository *Repository, storageDir string) *Service {
	return &Service{repository: repository, storageDir: storageDir}
}

func (s *Service) List(ctx context.Context, page, pageSize int) (ListPhotosResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	items, total, err := s.repository.List(ctx, pageSize, (page-1)*pageSize)
	if err != nil {
		return ListPhotosResponse{}, err
	}

	pages := 0
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
	}

	return ListPhotosResponse{
		Items:       items,
		Page:        page,
		PageSize:    pageSize,
		Pages:       pages,
		Total:       total,
		HasNext:     page < pages,
		HasPrevious: page > 1 && pages > 0,
	}, nil
}

func (s *Service) Get(ctx context.Context, id string) (Photo, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, source io.Reader, contentType string) (Photo, error) {
	id := newID()
	extension := extensionForContentType(contentType)
	if extension == "" {
		return Photo{}, ErrInvalidPhoto
	}

	if err := os.MkdirAll(s.storageDir, 0o755); err != nil {
		return Photo{}, fmt.Errorf("create photo storage directory: %w", err)
	}

	fileName := id + extension
	filePath := filepath.Join(s.storageDir, fileName)
	target, err := os.Create(filePath)
	if err != nil {
		return Photo{}, fmt.Errorf("create photo file: %w", err)
	}

	written, copyErr := io.Copy(target, io.LimitReader(source, MaxUploadBytes+1))
	closeErr := target.Close()
	if copyErr != nil {
		os.Remove(filePath)
		return Photo{}, fmt.Errorf("write photo file: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(filePath)
		return Photo{}, fmt.Errorf("close photo file: %w", closeErr)
	}
	if written == 0 || written > MaxUploadBytes {
		os.Remove(filePath)
		return Photo{}, ErrInvalidPhoto
	}

	width, height, err := imageDimensions(filePath)
	if err != nil {
		os.Remove(filePath)
		return Photo{}, ErrInvalidPhoto
	}

	photo := Photo{
		ID:          id,
		FileName:    fileName,
		ContentType: contentType,
		SizeBytes:   written,
		Width:       width,
		Height:      height,
		URL:         "/api/v1/photos/" + id + "/file",
		CreatedAt:   time.Now().UTC(),
	}

	created, err := s.repository.Create(ctx, photo)
	if err != nil {
		os.Remove(filePath)
		return Photo{}, err
	}

	return created, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	photo, err := s.repository.Delete(ctx, id)
	if err != nil {
		return err
	}

	if err := os.Remove(s.FilePath(photo)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete photo file: %w", err)
	}

	return nil
}

func (s *Service) FilePath(photo Photo) string {
	return filepath.Join(s.storageDir, filepath.Base(photo.FileName))
}

func imageDimensions(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return config.Width, config.Height, nil
}

func extensionForContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(bytes)
}
