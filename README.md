# HomeLab

HomeLab is a personal utilities app for small workflows that are useful at home, on a phone, tablet, or desktop.

The project is organized as a simple monorepo:

```text
HomeLab/
  frontend/   React + Vite app
  backend/    Go REST API
```

## Current Features

### Clipboard

Save reusable text snippets and copy them later with one click.

- Create snippets.
- Copy any saved item to the system clipboard.
- Delete one item or clear the full list.
- Paginated list, 15 items by default.
- Data is persisted in SQLite through the backend.

### Photos

Take photos from a device camera and keep them in a gallery.

- Opens the native camera on supported mobile browsers.
- Saves the original image file received from the browser, without frontend recompression.
- Shows a gallery of saved photos.
- Opens photos in a larger viewer.
- Deletes saved photos.
- Stores metadata in SQLite and image files on disk.

### Home Camera

View the live stream from a home camera.

- Dedicated page for the camera stream.
- The frontend reads the stream directly from `VITE_CAMERA_STREAM_URL`.
- Default stream URL: `http://192.168.31.67/stream`.

### Terminal

Open an interactive SSH terminal to a pre-configured host from a web page.

- Full terminal (xterm.js) with colors, control keys and resize.
- The browser talks WebSocket to the backend, which bridges an SSH PTY session.
- Target host and credentials are configured server-side; clients cannot pick a host.
- Disabled by default. Enable with `SSH_ENABLED=true` plus host/credentials.
- See [doc_ssh.md](./doc_ssh.md) for the design and full configuration reference.

## Tech Stack

Frontend:

```text
React
Vite
Sileo notifications
Nginx for production serving
```

Backend:

```text
Go
REST API
SQLite
modernc.org/sqlite
```

Infrastructure:

```text
Docker
Docker Compose
```

## Local Development

Run the backend:

```bash
cd backend
go run ./cmd/api
```

Run the frontend:

```bash
cd frontend
pnpm install
pnpm dev
```

Optional frontend environment:

```bash
cp frontend/.env.example frontend/.env
```

```text
VITE_CAMERA_STREAM_URL=http://192.168.31.67/stream
```

Default local URLs:

```text
Frontend: http://localhost:5173 or http://localhost:5174
Backend:  http://localhost:8080
```

## Docker Deployment

From the repository root:

```bash
docker compose up -d --build
```

To override the camera stream URL during Docker build:

```bash
VITE_CAMERA_STREAM_URL=http://192.168.31.67/stream docker compose up -d --build
```

Open:

```text
http://SERVER_IP:8081
```

The frontend container serves the React app with Nginx and proxies `/api/*` to the backend container through the Docker network.

Persistent data is stored on the host:

```text
backend/data/homelab.db
backend/data/photos/
```

Back up `backend/data/` to preserve clipboard snippets and photos.

More deployment details are in [DEPLOYMENT.md](./DEPLOYMENT.md).

## Validation

Frontend:

```bash
cd frontend
pnpm lint
pnpm build
```

Backend:

```bash
cd backend
go test ./...
go build ./cmd/api
```

Docker:

```bash
docker compose build
docker compose up -d
docker compose ps
```

## Notes

For mobile camera flows on a real server, use HTTPS in front of the app. Modern browsers are stricter with camera and device APIs outside secure contexts.
