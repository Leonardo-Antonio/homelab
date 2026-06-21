# HomeLab Docker Deployment

## Run

```bash
docker compose up -d --build
```

Open:

```text
http://SERVER_IP:8081
```

## Services

```text
frontend  Nginx + static React app  port 8081 -> 80
backend   Go REST API               internal port 8080
```

The frontend proxies `/api/*` to the backend through the internal Docker network.

## Persistent Data

SQLite and uploaded photos are stored on the host:

```text
backend/data/homelab.db
backend/data/photos/
```

Back up `backend/data/` to preserve clipboard snippets and photos.

## Useful Commands

```bash
docker compose ps
docker compose logs -f
docker compose logs -f backend
docker compose down
docker compose up -d --build
```

## Server Notes

If you expose the app publicly, put a reverse proxy with HTTPS in front of port `8081`.
HTTPS is recommended for mobile camera flows and generally required by modern browsers
for more advanced device APIs.
