package storage

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	TypeDir  = "dir"
	TypeFile = "file"

	MaxNameLength = 255
	// MaxUploadBytes caps a single file upload. Generous for a personal
	// homelab drive while still bounding disk and memory pressure.
	MaxUploadBytes = 5 << 30 // 5 GiB
)

var (
	ErrNotFound       = errors.New("node not found")
	ErrNotAFile       = errors.New("node is not a file")
	ErrInvalidName    = errors.New("name is required and must not exceed 255 characters")
	ErrInvalidParent  = errors.New("parent folder does not exist or is not a folder")
	ErrNameConflict   = errors.New("a node with that name already exists in this folder")
	ErrMoveIntoSelf   = errors.New("a folder cannot be moved into itself or one of its descendants")
	ErrEmptyUpload    = errors.New("uploaded file is empty")
	ErrUploadTooLarge = errors.New("uploaded file exceeds the maximum allowed size")
)

// Node is a single entry in the storage tree: either a folder ("dir") or a
// file. File-only fields (BlobID, ContentType, SizeBytes) are zero for folders.
type Node struct {
	ID          string    `json:"id"`
	ParentID    *string   `json:"parentId"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType,omitempty"`
	SizeBytes   int64     `json:"sizeBytes"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	// DownloadURL is set for files so the client can fetch the content.
	DownloadURL string `json:"downloadUrl,omitempty"`
	// ThumbnailURL is set for image files that can be previewed cheaply.
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	// blobID stays server-side; it is never serialized to clients.
	blobID string `json:"-"`
}

// Breadcrumb is a lightweight ancestor reference used to render the path bar.
type Breadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListResponse is the payload returned when browsing a folder.
type ListResponse struct {
	Parent     *Breadcrumb  `json:"parent"`
	Breadcrumb []Breadcrumb `json:"breadcrumb"`
	Items      []Node       `json:"items"`
}

type CreateFolderRequest struct {
	ParentID *string `json:"parentId"`
	Name     string  `json:"name"`
}

// UpdateRequest renames and/or moves a node. A nil field leaves that attribute
// untouched; ParentID is double-wrapped so "move to root" (explicit null) can
// be told apart from "do not move" (field absent):
//
//	ParentID == nil                 -> field absent: keep current parent
//	ParentID != nil, *ParentID == nil -> explicit null: move to root
//	ParentID != nil, *ParentID != nil -> move under that folder id
type UpdateRequest struct {
	Name     *string
	ParentID **string
}

// UnmarshalJSON distinguishes an absent "parentId" key from an explicit null.
// encoding/json collapses both onto a nil pointer for a plain **string field,
// so we decode the key as RawMessage and inspect whether it was present.
func (r *UpdateRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name     *string         `json:"name"`
		ParentID json.RawMessage `json:"parentId"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Name = raw.Name
	r.ParentID = nil
	if raw.ParentID != nil { // key was present (either null or a string)
		var inner *string
		if err := json.Unmarshal(raw.ParentID, &inner); err != nil {
			return err
		}
		r.ParentID = &inner
	}
	return nil
}
