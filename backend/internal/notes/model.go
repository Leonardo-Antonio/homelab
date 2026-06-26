package notes

import (
	"errors"
	"time"
)

const (
	TypeDir          = "dir"
	TypeNote         = "note"
	MaxNameLength    = 255
	MaxContentLength = 2_000_000
)

var (
	ErrNotFound       = errors.New("note not found")
	ErrInvalidName    = errors.New("name is required and must not exceed 255 characters")
	ErrInvalidType    = errors.New("type must be 'dir' or 'note'")
	ErrContentTooLong = errors.New("content exceeds the maximum allowed length")
)

type Node struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parentId"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type NoteDetail struct {
	Node
	Content string `json:"content"`
}

type CreateRequest struct {
	ParentID *string `json:"parentId"`
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Content  string  `json:"content"`
}

// UpdateRequest uses pointer semantics: nil ParentID means move to root,
// non-nil means move to that parent. Name and Content always replace current values.
type UpdateRequest struct {
	Name     string  `json:"name"`
	Content  string  `json:"content"`
	ParentID *string `json:"parentId"`
}
