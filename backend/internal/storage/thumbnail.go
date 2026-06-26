package storage

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// thumbMaxEdge is the longest side (px) of a generated thumbnail. Small enough
// to make a grid of previews load instantly, large enough to look crisp on
// retina screens.
const thumbMaxEdge = 360

// thumbMaxPixels guards against decompression bombs: images whose declared
// dimensions exceed this are refused before the full pixel buffer is decoded.
const thumbMaxPixels = 64 * 1000 * 1000 // 64 MP

var errNotThumbnailable = errors.New("content type cannot be thumbnailed")

// thumbnailer generates and caches downscaled JPEG previews for image blobs.
// Thumbnails are keyed by the blob digest, so they are produced once per unique
// image and shared across every file that points at the same content.
type thumbnailer struct {
	dir string
}

func newThumbnailer(root string) (*thumbnailer, error) {
	dir := filepath.Join(root, "thumbs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create thumbnail dir: %w", err)
	}
	return &thumbnailer{dir: dir}, nil
}

// thumbnailable reports whether a content type is a raster image we can decode.
func thumbnailable(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func (t *thumbnailer) path(digest string) string {
	return filepath.Join(t.dir, digest[:2], digest+".jpg")
}

// Get returns the path to a cached thumbnail for the blob at sourcePath,
// generating it on first request. The generated file is written atomically so a
// concurrent reader never sees a partial image.
func (t *thumbnailer) Get(digest, sourcePath, contentType string) (string, error) {
	if !thumbnailable(contentType) {
		return "", errNotThumbnailable
	}

	dest := t.path(digest)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil // cache hit
	}

	if err := t.generate(sourcePath, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// Remove deletes a cached thumbnail (best effort; a miss is not an error) so
// thumbnail cleanup can ride along with blob garbage collection.
func (t *thumbnailer) Remove(digest string) {
	_ = os.Remove(t.path(digest))
}

func (t *thumbnailer) generate(sourcePath, dest string) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source for thumbnail: %w", err)
	}
	defer src.Close()

	// Reject oversized images before allocating their full pixel buffer.
	cfg, _, err := image.DecodeConfig(src)
	if err != nil {
		return errNotThumbnailable
	}
	if int64(cfg.Width)*int64(cfg.Height) > thumbMaxPixels {
		return errNotThumbnailable
	}
	if _, err := src.Seek(0, 0); err != nil {
		return fmt.Errorf("rewind source: %w", err)
	}

	img, _, err := image.Decode(src)
	if err != nil {
		return errNotThumbnailable
	}

	thumb := scaleToFit(img, thumbMaxEdge)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create thumbnail shard: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), "thumb-*")
	if err != nil {
		return fmt.Errorf("create thumbnail temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := jpeg.Encode(tmp, thumb, &jpeg.Options{Quality: 82}); err != nil {
		tmp.Close()
		return fmt.Errorf("encode thumbnail: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync thumbnail: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close thumbnail: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("commit thumbnail: %w", err)
	}
	return nil
}

// scaleToFit downscales img so its longest edge is at most maxEdge, preserving
// aspect ratio. Images already within bounds are returned untouched.
func scaleToFit(img image.Image, maxEdge int) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxEdge && h <= maxEdge {
		return img
	}

	scale := float64(maxEdge) / float64(w)
	if h > w {
		scale = float64(maxEdge) / float64(h)
	}
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}
