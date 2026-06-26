# HomeLab Backend

REST API for HomeLab.

## Run

```bash
go run ./cmd/api
```

Default values:

```text
HTTP_ADDR=:8080
DATABASE_PATH=data/homelab.db
PHOTO_STORAGE_DIR=data/photos
STORAGE_DIR=data/storage
ALLOWED_ORIGIN=http://localhost:5174
```

## Clipboard API

```text
GET    /healthz
GET    /api/v1/clipboard-items?page=1&pageSize=15
POST   /api/v1/clipboard-items
GET    /api/v1/clipboard-items/{id}
DELETE /api/v1/clipboard-items/{id}
DELETE /api/v1/clipboard-items
```

Create body:

```json
{
  "text": "snippet text"
}
```

## Photos API

```text
GET    /api/v1/photos?page=1&pageSize=15
POST   /api/v1/photos
GET    /api/v1/photos/{id}
GET    /api/v1/photos/{id}/file
DELETE /api/v1/photos/{id}
```

Create uses `multipart/form-data` with a `photo` file field. JPEG and PNG files up to 8MB are accepted.

## Storage API (Drive-like)

A reliable, content-addressed file store: folders and files in a tree, backed by
on-disk blobs keyed by their SHA-256 digest.

```text
GET    /api/v1/storage/nodes?parentId={id}      # list a folder (omit parentId for root)
GET    /api/v1/storage/nodes/{id}               # get a single node
POST   /api/v1/storage/folders                  # { "parentId": null, "name": "Docs" }
POST   /api/v1/storage/files                    # multipart: parentId, file
GET    /api/v1/storage/files/{id}/content       # download (supports Range; ?download=1 forces attachment)
PATCH  /api/v1/storage/nodes/{id}               # rename and/or move: { "name": "...", "parentId": "..." }
DELETE /api/v1/storage/nodes/{id}               # delete a file, or a folder and its whole subtree
```

Reliability guarantees:

- **Atomic uploads** — content is streamed to a temp file, `fsync`'d, then
  atomically renamed into place (directory also `fsync`'d). A partial upload is
  never visible as a real file.
- **Durable content before metadata** — the blob is on disk before the database
  row is committed; a failed metadata write garbage-collects the orphan blob.
- **Deduplication + ref counting** — identical content is stored once; a blob is
  only deleted from disk once no node references it (ref counts are maintained
  transactionally by SQLite triggers, exact even across cascading deletes).
- **Startup sweep** — any blob left unreferenced by a crash is reclaimed on boot.

Single files up to 5 GiB are accepted (`MaxUploadBytes`).
