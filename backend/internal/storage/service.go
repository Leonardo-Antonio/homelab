package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"
)

type Service struct {
	repo   *Repository
	blobs  *blobStore
	thumbs *thumbnailer
}

func NewService(repo *Repository, storageDir string) (*Service, error) {
	blobs, err := newBlobStore(storageDir)
	if err != nil {
		return nil, err
	}
	thumbs, err := newThumbnailer(storageDir)
	if err != nil {
		return nil, err
	}
	s := &Service{repo: repo, blobs: blobs, thumbs: thumbs}
	s.sweepOrphans(context.Background())
	return s, nil
}

// List returns a folder's children plus the breadcrumb trail leading to it.
func (s *Service) List(ctx context.Context, parentID *string) (ListResponse, error) {
	var breadcrumb []Breadcrumb
	var parent *Breadcrumb
	if parentID != nil {
		trail, err := s.repo.Ancestors(ctx, *parentID)
		if err != nil {
			return ListResponse{}, err
		}
		breadcrumb = trail
		if len(trail) > 0 {
			parent = &trail[len(trail)-1]
		}
	}

	items, err := s.repo.ListChildren(ctx, parentID)
	if err != nil {
		return ListResponse{}, err
	}
	for i := range items {
		decorate(&items[i])
	}

	return ListResponse{Parent: parent, Breadcrumb: breadcrumb, Items: items}, nil
}

// Get returns a single node decorated with its download URL.
func (s *Service) Get(ctx context.Context, id string) (Node, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return Node{}, err
	}
	decorate(&node)
	return node, nil
}

func (s *Service) CreateFolder(ctx context.Context, req CreateFolderRequest) (Node, error) {
	name, err := sanitizeName(req.Name)
	if err != nil {
		return Node{}, err
	}
	now := time.Now().UTC()
	node := Node{
		ID:        newID(),
		ParentID:  normalizeParent(req.ParentID),
		Type:      TypeDir,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.CreateFolder(ctx, node)
}

// CreateFile durably stores the uploaded content first, then records metadata.
// On any metadata failure the freshly written blob is swept so no orphan
// lingers. The name is auto-deduplicated within the target folder so an upload
// never fails merely because the name is taken (Drive-style "(2)" suffixing).
func (s *Service) CreateFile(ctx context.Context, parentID *string, name, contentType string, content io.Reader) (Node, error) {
	parentID = normalizeParent(parentID)

	cleanName, err := sanitizeName(name)
	if err != nil {
		return Node{}, err
	}

	// 1. Durable content before metadata.
	res, err := s.blobs.Write(content)
	if err != nil {
		return Node{}, err
	}

	now := time.Now().UTC()
	node := Node{
		ID:          newID(),
		ParentID:    parentID,
		Type:        TypeFile,
		Name:        cleanName,
		ContentType: strings.TrimSpace(contentType),
		SizeBytes:   res.Size,
		CreatedAt:   now,
		UpdatedAt:   now,
		blobID:      res.Digest,
	}

	// 2. Insert metadata, retrying with a deduplicated name on collision.
	for attempt := 0; attempt < 64; attempt++ {
		created, err := s.repo.CreateFile(ctx, node, res.Size)
		if err == nil {
			decorate(&created)
			return created, nil
		}
		if err == ErrNameConflict {
			node.Name = dedupeName(cleanName, attempt+2)
			continue
		}
		// Metadata failed for another reason: roll back the blob if it is now
		// unreferenced, so a failed upload leaves nothing behind.
		s.sweepOrphans(ctx)
		return Node{}, err
	}
	s.sweepOrphans(ctx)
	return Node{}, ErrNameConflict
}

// OpenFile resolves a file node and returns an io.ReadSeekCloser over its
// content, suitable for http.ServeContent (range requests, caching).
func (s *Service) OpenFile(ctx context.Context, id string) (Node, *os.File, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return Node{}, nil, err
	}
	if node.Type != TypeFile || node.blobID == "" {
		return Node{}, nil, ErrNotAFile
	}
	f, err := os.Open(s.blobs.path(node.blobID))
	if err != nil {
		return Node{}, nil, fmt.Errorf("open blob: %w", err)
	}
	return node, f, nil
}

// OpenThumbnail returns a small cached JPEG preview for an image file,
// generating it lazily on first request. Non-image files yield ErrNotAFile so
// the handler can answer 404.
func (s *Service) OpenThumbnail(ctx context.Context, id string) (*os.File, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if node.Type != TypeFile || node.blobID == "" || !thumbnailable(node.ContentType) {
		return nil, ErrNotAFile
	}
	path, err := s.thumbs.Get(node.blobID, s.blobs.path(node.blobID), node.ContentType)
	if err != nil {
		if err == errNotThumbnailable {
			return nil, ErrNotAFile
		}
		return nil, fmt.Errorf("thumbnail: %w", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open thumbnail: %w", err)
	}
	return f, nil
}

// Update renames and/or moves a node, guarding against moving a folder into
// its own subtree.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Node, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Node{}, err
	}

	name := current.Name
	if req.Name != nil {
		if name, err = sanitizeName(*req.Name); err != nil {
			return Node{}, err
		}
	}

	parentID := current.ParentID
	if req.ParentID != nil {
		parentID = normalizeParent(*req.ParentID)
		if current.Type == TypeDir && parentID != nil {
			if err := s.assertNotDescendant(ctx, id, *parentID); err != nil {
				return Node{}, err
			}
		}
	}

	updated, err := s.repo.Update(ctx, id, name, parentID, time.Now().UTC())
	if err != nil {
		return Node{}, err
	}
	decorate(&updated)
	return updated, nil
}

// Delete removes a node and its subtree, then garbage-collects any blobs that
// became unreferenced.
func (s *Service) Delete(ctx context.Context, id string) error {
	orphans, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	s.removeBlobs(orphans)
	return nil
}

// ── internals ───────────────────────────────────────────────────────────────

// assertNotDescendant fails if targetParent is id itself or lives inside id.
func (s *Service) assertNotDescendant(ctx context.Context, id, targetParent string) error {
	if id == targetParent {
		return ErrMoveIntoSelf
	}
	trail, err := s.repo.Ancestors(ctx, targetParent)
	if err != nil {
		return err
	}
	for _, ancestor := range trail {
		if ancestor.ID == id {
			return ErrMoveIntoSelf
		}
	}
	return nil
}

func (s *Service) sweepOrphans(ctx context.Context) {
	orphans, err := s.repo.SweepOrphans(ctx)
	if err != nil {
		slog.Error("storage orphan sweep failed", "error", err)
		return
	}
	s.removeBlobs(orphans)
}

func (s *Service) removeBlobs(digests []string) {
	for _, digest := range digests {
		if err := s.blobs.Remove(digest); err != nil {
			slog.Error("storage blob removal failed", "digest", digest, "error", err)
		}
		// A blob and its derived thumbnail share a lifetime.
		s.thumbs.Remove(digest)
	}
}

func decorate(node *Node) {
	if node.Type != TypeFile {
		return
	}
	node.DownloadURL = "/api/v1/storage/files/" + node.ID + "/content"
	if thumbnailable(node.ContentType) {
		node.ThumbnailURL = "/api/v1/storage/files/" + node.ID + "/thumbnail"
	}
}

func normalizeParent(p *string) *string {
	if p == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*p)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func sanitizeName(name string) (string, error) {
	// Collapse to the base name so a client can never smuggle path separators
	// (e.g. "../../etc") into a stored name.
	clean := strings.TrimSpace(path.Base(strings.ReplaceAll(name, "\\", "/")))
	clean = strings.TrimRight(clean, ". ")
	if clean == "" || clean == "." || clean == ".." || len(clean) > MaxNameLength {
		return "", ErrInvalidName
	}
	return clean, nil
}

// dedupeName turns "report.pdf" into "report (2).pdf", keeping the extension.
func dedupeName(name string, n int) string {
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s (%d)%s", base, n, ext)
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}
