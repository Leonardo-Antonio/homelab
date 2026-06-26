package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"homelab/backend/internal/database"
)

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	svc, err := NewService(NewRepository(db), filepath.Join(dir, "storage"))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc, dir
}

func upload(t *testing.T, svc *Service, parentID *string, name, content string) Node {
	t.Helper()
	node, err := svc.CreateFile(context.Background(), parentID, name, "text/plain", strings.NewReader(content))
	if err != nil {
		t.Fatalf("upload %q: %v", name, err)
	}
	return node
}

func TestUploadDownloadRoundTrip(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	node := upload(t, svc, nil, "hello.txt", "hello world")
	if node.SizeBytes != 11 {
		t.Fatalf("size = %d, want 11", node.SizeBytes)
	}
	if node.DownloadURL == "" {
		t.Fatal("expected a download URL")
	}

	_, f, err := svc.OpenFile(ctx, node.ID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if string(got) != "hello world" {
		t.Fatalf("content = %q, want %q", got, "hello world")
	}
}

func TestDeduplicationSharesBlob(t *testing.T) {
	svc, root := newTestService(t)

	a := upload(t, svc, nil, "a.txt", "same bytes")
	b := upload(t, svc, nil, "b.txt", "same bytes")
	if a.blobID != b.blobID {
		t.Fatal("identical content should share a blob")
	}

	// Exactly one physical blob on disk despite two logical files.
	count := countBlobFiles(t, root)
	if count != 1 {
		t.Fatalf("blob files on disk = %d, want 1", count)
	}
}

func TestDeleteGarbageCollectsUnreferencedBlob(t *testing.T) {
	svc, root := newTestService(t)
	ctx := context.Background()

	a := upload(t, svc, nil, "a.txt", "unique-a")
	b := upload(t, svc, nil, "b.txt", "unique-b")

	if err := svc.Delete(ctx, a.ID); err != nil {
		t.Fatalf("delete a: %v", err)
	}
	if got := countBlobFiles(t, root); got != 1 {
		t.Fatalf("after deleting a, blob files = %d, want 1", got)
	}
	if err := svc.Delete(ctx, b.ID); err != nil {
		t.Fatalf("delete b: %v", err)
	}
	if got := countBlobFiles(t, root); got != 0 {
		t.Fatalf("after deleting both, blob files = %d, want 0", got)
	}
}

func TestSharedBlobSurvivesUntilLastReference(t *testing.T) {
	svc, root := newTestService(t)
	ctx := context.Background()

	a := upload(t, svc, nil, "a.txt", "shared")
	upload(t, svc, nil, "b.txt", "shared")

	if err := svc.Delete(ctx, a.ID); err != nil {
		t.Fatalf("delete a: %v", err)
	}
	// b still references the blob: it must remain on disk and be readable.
	if got := countBlobFiles(t, root); got != 1 {
		t.Fatalf("shared blob removed too early: files = %d, want 1", got)
	}
}

func TestDeleteFolderCascades(t *testing.T) {
	svc, root := newTestService(t)
	ctx := context.Background()

	folder, err := svc.CreateFolder(ctx, CreateFolderRequest{Name: "docs"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	upload(t, svc, &folder.ID, "nested.txt", "nested content")

	if err := svc.Delete(ctx, folder.ID); err != nil {
		t.Fatalf("delete folder: %v", err)
	}

	list, err := svc.List(ctx, nil)
	if err != nil {
		t.Fatalf("list root: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("root items = %d, want 0", len(list.Items))
	}
	if got := countBlobFiles(t, root); got != 0 {
		t.Fatalf("blob files after cascade = %d, want 0", got)
	}
}

func TestUploadAutoDedupesName(t *testing.T) {
	svc, _ := newTestService(t)

	first := upload(t, svc, nil, "report.pdf", "v1")
	second := upload(t, svc, nil, "report.pdf", "v2")

	if first.Name != "report.pdf" {
		t.Fatalf("first name = %q", first.Name)
	}
	if second.Name != "report (2).pdf" {
		t.Fatalf("second name = %q, want %q", second.Name, "report (2).pdf")
	}
}

func TestCreateFolderRejectsDuplicateName(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateFolder(ctx, CreateFolderRequest{Name: "shared"}); err != nil {
		t.Fatalf("first folder: %v", err)
	}
	_, err := svc.CreateFolder(ctx, CreateFolderRequest{Name: "shared"})
	if err != ErrNameConflict {
		t.Fatalf("err = %v, want ErrNameConflict", err)
	}
}

func TestMoveFileToRoot(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	folder, _ := svc.CreateFolder(ctx, CreateFolderRequest{Name: "box"})
	file := upload(t, svc, &folder.ID, "doc.txt", "data")

	// Explicit null parentId must move the file back to the root.
	var req UpdateRequest
	if err := json.Unmarshal([]byte(`{"parentId":null}`), &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	moved, err := svc.Update(ctx, file.ID, req)
	if err != nil {
		t.Fatalf("move to root: %v", err)
	}
	if moved.ParentID != nil {
		t.Fatalf("parentId = %v, want nil (root)", *moved.ParentID)
	}

	root, _ := svc.List(ctx, nil)
	names := map[string]bool{}
	for _, n := range root.Items {
		names[n.Name] = true
	}
	if !names["doc.txt"] {
		t.Fatalf("file not at root after move: %+v", root.Items)
	}
}

func TestUpdateRequestDistinguishesAbsentFromNullParent(t *testing.T) {
	cases := []struct {
		body          string
		wantOuterNil  bool // ParentID == nil  (absent: keep current)
		wantInnerNil  bool // *ParentID == nil (explicit null: root)
		wantInnerText string
	}{
		{`{"name":"x"}`, true, false, ""},        // absent
		{`{"parentId":null}`, false, true, ""},   // explicit null
		{`{"parentId":"abc"}`, false, false, "abc"}, // a folder id
	}
	for _, tc := range cases {
		var req UpdateRequest
		if err := json.Unmarshal([]byte(tc.body), &req); err != nil {
			t.Fatalf("%s: %v", tc.body, err)
		}
		if (req.ParentID == nil) != tc.wantOuterNil {
			t.Fatalf("%s: outer nil = %v, want %v", tc.body, req.ParentID == nil, tc.wantOuterNil)
		}
		if tc.wantOuterNil {
			continue
		}
		if (*req.ParentID == nil) != tc.wantInnerNil {
			t.Fatalf("%s: inner nil = %v, want %v", tc.body, *req.ParentID == nil, tc.wantInnerNil)
		}
		if !tc.wantInnerNil && **req.ParentID != tc.wantInnerText {
			t.Fatalf("%s: inner = %q, want %q", tc.body, **req.ParentID, tc.wantInnerText)
		}
	}
}

func TestMoveFolderIntoDescendantRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	parent, _ := svc.CreateFolder(ctx, CreateFolderRequest{Name: "parent"})
	child, _ := svc.CreateFolder(ctx, CreateFolderRequest{Name: "child", ParentID: &parent.ID})

	childID := child.ID
	_, err := svc.Update(ctx, parent.ID, UpdateRequest{ParentID: ptrPtr(&childID)})
	if err != ErrMoveIntoSelf {
		t.Fatalf("err = %v, want ErrMoveIntoSelf", err)
	}
}

func TestRejectsPathTraversalName(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.CreateFolder(context.Background(), CreateFolderRequest{Name: "../../etc"})
	// path.Base reduces "../../etc" to "etc", so it is accepted but sanitized.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, _ := svc.List(context.Background(), nil)
	if len(list.Items) != 1 || list.Items[0].Name != "etc" {
		t.Fatalf("name not sanitized: %+v", list.Items)
	}
}

func TestThumbnailGeneratedAndCleanedUp(t *testing.T) {
	svc, root := newTestService(t)
	ctx := context.Background()

	// A 600x400 PNG — larger than thumbMaxEdge, so it must be downscaled.
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 600, 400))
	for x := 0; x < 600; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	node, err := svc.CreateFile(ctx, nil, "pic.png", "image/png", bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("upload image: %v", err)
	}
	if node.ThumbnailURL == "" {
		t.Fatal("image file should expose a thumbnail URL")
	}

	f, err := svc.OpenThumbnail(ctx, node.ID)
	if err != nil {
		t.Fatalf("open thumbnail: %v", err)
	}
	thumb, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	if b := thumb.Bounds(); b.Dx() > thumbMaxEdge || b.Dy() > thumbMaxEdge {
		t.Fatalf("thumbnail not downscaled: %dx%d", b.Dx(), b.Dy())
	}
	if countThumbFiles(t, root) != 1 {
		t.Fatalf("expected exactly one cached thumbnail")
	}

	// Deleting the file must reclaim both the blob and its thumbnail.
	if err := svc.Delete(ctx, node.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if countThumbFiles(t, root) != 0 {
		t.Fatalf("thumbnail not cleaned up after delete")
	}
}

func TestThumbnailRejectsNonImage(t *testing.T) {
	svc, _ := newTestService(t)
	node := upload(t, svc, nil, "notes.txt", "just text")
	if node.ThumbnailURL != "" {
		t.Fatal("text file should not advertise a thumbnail")
	}
	if _, err := svc.OpenThumbnail(context.Background(), node.ID); err != ErrNotAFile {
		t.Fatalf("err = %v, want ErrNotAFile", err)
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func ptrPtr(p *string) **string { return &p }

func countThumbFiles(t *testing.T, root string) int {
	t.Helper()
	return countFilesUnder(t, filepath.Join(root, "storage", "thumbs"))
}

func countBlobFiles(t *testing.T, root string) int {
	t.Helper()
	return countFilesUnder(t, filepath.Join(root, "storage", "blobs"))
}

func countFilesUnder(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk blobs: %v", err)
	}
	return count
}
