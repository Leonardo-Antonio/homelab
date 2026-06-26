package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// blobStore persists file contents on disk addressed by their SHA-256 digest.
//
// Layout:  <root>/blobs/<ab>/<abcdef...>   (sharded by the first byte)
//          <root>/tmp/<random>            (staging area for atomic writes)
//
// Writes are crash-safe: content is streamed to a temp file, fsync'd, then
// atomically renamed into place and the destination directory is fsync'd too.
// A partially written upload can therefore never be observed as a real blob.
type blobStore struct {
	root string
}

func newBlobStore(root string) (*blobStore, error) {
	for _, dir := range []string{filepath.Join(root, "blobs"), filepath.Join(root, "tmp")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage directory %q: %w", dir, err)
		}
	}
	return &blobStore{root: root}, nil
}

func (b *blobStore) path(digest string) string {
	return filepath.Join(b.root, "blobs", digest[:2], digest)
}

// Has reports whether the physical blob for digest exists on disk.
func (b *blobStore) Has(digest string) bool {
	_, err := os.Stat(b.path(digest))
	return err == nil
}

// writeResult carries the outcome of streaming an upload to disk.
type writeResult struct {
	Digest string
	Size   int64
}

// Write streams src into a temp file (capped at MaxUploadBytes), computing the
// SHA-256 digest as it goes, then durably moves it to its content-addressed
// location. If a blob with the same digest already exists, the temp file is
// discarded and the existing blob is reused (deduplication).
func (b *blobStore) Write(src io.Reader) (writeResult, error) {
	tmp, err := os.CreateTemp(filepath.Join(b.root, "tmp"), "upload-*")
	if err != nil {
		return writeResult{}, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup; on the success path the file is renamed away first.
	defer os.Remove(tmpName)

	hasher := sha256.New()
	limited := io.LimitReader(src, MaxUploadBytes+1)
	size, err := io.Copy(io.MultiWriter(tmp, hasher), limited)
	if err != nil {
		tmp.Close()
		return writeResult{}, fmt.Errorf("write temp file: %w", err)
	}
	if size > MaxUploadBytes {
		tmp.Close()
		return writeResult{}, ErrUploadTooLarge
	}
	if size == 0 {
		tmp.Close()
		return writeResult{}, ErrEmptyUpload
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return writeResult{}, fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return writeResult{}, fmt.Errorf("close temp file: %w", err)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	dest := b.path(digest)

	// Reuse an identical blob if one is already present.
	if b.Has(digest) {
		return writeResult{Digest: digest, Size: size}, nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return writeResult{}, fmt.Errorf("create blob shard dir: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return writeResult{}, fmt.Errorf("commit blob: %w", err)
	}
	if err := syncDir(filepath.Dir(dest)); err != nil {
		return writeResult{}, fmt.Errorf("sync blob dir: %w", err)
	}

	return writeResult{Digest: digest, Size: size}, nil
}

// Remove deletes the physical blob for digest. A missing file is not an error
// so garbage collection is idempotent.
func (b *blobStore) Remove(digest string) error {
	if err := os.Remove(b.path(digest)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove blob: %w", err)
	}
	return nil
}

// syncDir flushes a directory entry change (a rename) to stable storage.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
