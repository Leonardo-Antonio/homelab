package storage

import (
	"context"
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

// ── helpers ─────────────────────────────────────────────────────────────────

func ptrPtr(p *string) **string { return &p }

func countBlobFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	blobsDir := filepath.Join(root, "storage", "blobs")
	err := filepath.Walk(blobsDir, func(_ string, info os.FileInfo, err error) error {
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
