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
