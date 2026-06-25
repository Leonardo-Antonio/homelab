# Web SSH Terminal — Design & Implementation

This document describes how the **SSH Terminal** feature is added to HomeLab: an
interactive terminal rendered on a frontend page that proxies an SSH session to a
pre-configured host through the Go backend.

The design follows the conventions already present in the repository:

- Backend: Go standard library `net/http` + `ServeMux`, clean separation in
  `internal/<feature>` (handler / service-like session / model), config from env.
- Frontend: React + Vite, page-per-feature under `src/pages`, thin API/transport
  modules under `src/services`, reusable logic in `src/hooks`, `sileo` toasts.

---

## 1. Goal

Provide a page (`Terminal`) inside the existing app shell where the user can run
commands on a remote machine over SSH, exactly like a native terminal: live
output, keystrokes, control sequences (Ctrl+C, arrows), colors and resizing.

---

## 2. Architecture

```text
┌────────────────────┐        WebSocket          ┌──────────────────────┐        SSH         ┌──────────────┐
│  Browser            │  /api/v1/terminal/ws      │  Go backend           │   x/crypto/ssh    │  Target host  │
│  xterm.js terminal  │ ───────────────────────▶  │  terminal.Handler     │ ────────────────▶ │  sshd + PTY   │
│  (TerminalPage)     │ ◀───────────────────────  │  ↔ SSH session (PTY)  │ ◀──────────────── │              │
└────────────────────┘   text (in)/binary (out)   └──────────────────────┘                    └──────────────┘
```

- The browser never talks SSH. It only speaks WebSocket to our backend.
- The backend opens **one SSH session per WebSocket connection** to a host that is
  configured server-side (host, port, user, credentials). Secrets stay on the
  server; the client cannot choose an arbitrary target (no open SSH proxy / SSRF).
- A PTY is requested on the SSH channel so interactive programs behave correctly.

### Why these libraries

| Concern        | Choice                          | Reason                                                         |
| -------------- | ------------------------------- | ------------------------------------------------------------- |
| WebSocket (Go) | `github.com/coder/websocket`    | Minimal, modern, context-aware, stdlib-style API.             |
| SSH client     | `golang.org/x/crypto/ssh`       | Canonical Go SSH implementation, no CGO.                      |
| Terminal (web) | `@xterm/xterm` + `addon-fit`    | De-facto standard browser terminal emulator, used everywhere. |

---

## 3. WebSocket sub-protocol

A tiny, explicit protocol keeps it maintainable and avoids guessing frame intent.

**Client → Server** (JSON text frames):

```jsonc
{ "type": "stdin",  "data": "ls -la\n" }     // raw keystrokes typed by the user
{ "type": "resize", "cols": 120, "rows": 32 } // terminal viewport changed
```

**Server → Client**:

- **Binary frames** = raw PTY output bytes, written straight into xterm
  (`term.write(uint8array)`). Binary avoids breaking multi-byte UTF-8 across frames.
- **Text frames (JSON)** = out-of-band status, e.g. connection lifecycle / errors:

  ```jsonc
  { "type": "status", "state": "connected" }
  { "type": "error",  "message": "ssh handshake failed" }
  ```

The client distinguishes them by type: `ArrayBuffer` → terminal output, `string`
→ JSON status. The WebSocket close code/reason carries the final reason when the
SSH session ends.

---

## 4. Backend changes

### 4.1 New config (`internal/config/config.go`)

New env vars (all optional except when you want a usable target):

| Env var                       | Default       | Meaning                                            |
| ----------------------------- | ------------- | -------------------------------------------------- |
| `SSH_ENABLED`                 | `false`       | Master switch for the feature.                     |
| `SSH_HOST`                    | `127.0.0.1`   | Target SSH host.                                   |
| `SSH_PORT`                    | `22`          | Target SSH port.                                   |
| `SSH_USER`                    | `""`          | SSH username.                                      |
| `SSH_PASSWORD`                | `""`          | Password auth (used if no key).                    |
| `SSH_PRIVATE_KEY_PATH`        | `""`          | Path to a private key (preferred over password).   |
| `SSH_PRIVATE_KEY_PASSPHRASE`  | `""`          | Passphrase for an encrypted private key.           |
| `SSH_KNOWN_HOSTS_PATH`        | `""`          | `known_hosts` file for host-key verification.      |
| `SSH_INSECURE_IGNORE_HOST_KEY`| `false`       | Skip host-key verification (homelab convenience).  |
| `SSH_CONNECT_TIMEOUT_SECONDS` | `10`          | SSH dial/handshake timeout.                        |

Host-key policy: if `SSH_KNOWN_HOSTS_PATH` is set it is enforced; otherwise, if
`SSH_INSECURE_IGNORE_HOST_KEY=true` verification is skipped (logged as a warning);
otherwise the connection is refused. This keeps the secure path the default.

### 4.2 New package `internal/terminal`

- `model.go` — protocol message structs/constants, `Config` for the SSH target.
- `ssh.go` — `dial(ctx, cfg)`: builds `*ssh.ClientConfig` (auth + host-key
  callback) and returns a connected `*ssh.Client`.
- `session.go` — `runSession`: requests a PTY + shell, then bridges:
  - goroutine A: read JSON frames from WS → write stdin / send window-change on resize.
  - goroutine B: read SSH stdout/stderr → write binary frames to WS.
  Closes both sides cleanly when either ends or the request context is cancelled.
- `handler.go` — `Handler` with `Register(mux)` exposing
  `GET /api/v1/terminal/ws`. It accepts the WS upgrade (origin-checked using the
  same `ALLOWED_ORIGIN` config), enforces `SSH_ENABLED`, dials SSH, and runs the
  session. A `GET /api/v1/terminal/info` endpoint reports `{enabled, host, user}`
  (no secrets) so the frontend can show target info and disable the page when off.

### 4.3 Wiring (`cmd/api/main.go`)

Construct `terminal.NewHandler(cfg.Terminal, cfg.AllowedOrigin)` and call
`Register(mux)` alongside the existing handlers. WebSocket routes are exempt from
the normal write timeout, so the long-lived `/api/v1/terminal/ws` connection is
created with its own context rather than relying on `WriteTimeout` (handled inside
the handler via hijacked conn semantics of `coder/websocket`).

> Note: the server's `WriteTimeout` does not abort an already-upgraded WebSocket
> because `coder/websocket` takes over the connection; long sessions are fine.

---

## 5. Frontend changes

### 5.1 Transport — `src/services/terminalSocket.js`

- Builds the WS URL from `VITE_API_BASE_URL` (same-origin in production via the
  Nginx proxy; configurable for dev where frontend `:5173` ≠ backend `:8080`).
  `http→ws`, `https→wss`.
- `fetchTerminalInfo()` calls `GET /api/v1/terminal/info`.
- `openTerminalSocket({ onOutput, onStatus, onClose })` returns
  `{ sendInput, sendResize, close }`, encapsulating the sub-protocol.

### 5.2 Hook — `src/hooks/useTerminalSession.js`

Owns the xterm instance lifecycle and the socket:

- Creates the `Terminal` + `FitAddon`, mounts into a container ref.
- Wires xterm `onData` → `sendInput`, `ResizeObserver` → `fit()` + `sendResize`.
- Exposes `{ containerRef, status, info, reconnect }`.
- Cleans up terminal, observer and socket on unmount (StrictMode-safe).

### 5.3 Page — `src/pages/TerminalPage/TerminalPage.jsx` (+ `.css`)

Follows the `CameraStreamPage` layout: header with eyebrow/title + a status pill
(`Conectando` / `Conectado` / `Desconectado`), a console panel hosting the xterm
container, and a reconnect button. Shows a clear message when `SSH_ENABLED` is off.

### 5.4 App integration

- `src/App.jsx`: add `terminal: <TerminalPage />` to the `pages` map.
- `src/components/AppShell.jsx`: add a nav item `{ id: 'terminal', label: 'Terminal', icon: '_' }`.
- `src/services/notifications.js`: add `terminalConnected` / `terminalDisconnected` / `terminalFailed`.

### 5.5 Nginx (`frontend/nginx.conf`)

The `/api/` proxy block gains the WebSocket upgrade headers so the long-lived
connection survives:

```nginx
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection $connection_upgrade;
```

(`$connection_upgrade` is mapped at the `http{}` level via a `map` so normal
requests get `close` and upgrade requests get `upgrade`.)

---

## 6. Security notes

- The target host and credentials are **server-side only**; clients cannot point
  the backend at arbitrary hosts.
- Host-key verification is on by default; the insecure skip is opt-in for homelab.
- The feature is **disabled by default** (`SSH_ENABLED=false`).
- Origin of the WebSocket is validated against `ALLOWED_ORIGIN`.
- For public exposure, terminate TLS in front (so `wss://`) and put auth in front
  of the app — the terminal grants real shell access.

---

## 7. Configuration example (`docker-compose.yml`)

```yaml
backend:
  environment:
    SSH_ENABLED: "true"
    SSH_HOST: "192.168.31.10"
    SSH_PORT: "22"
    SSH_USER: "leonardo"
    SSH_PASSWORD: "change-me"          # or mount a key and use SSH_PRIVATE_KEY_PATH
    SSH_INSECURE_IGNORE_HOST_KEY: "true"
```

For dev, point the frontend at the backend WebSocket:

```bash
# frontend/.env
VITE_API_BASE_URL=http://localhost:8080
```

---

## 8. Validation

```bash
# backend
cd backend && go build ./... && go vet ./... && go test ./...

# frontend
cd frontend && pnpm lint && pnpm build

# docker
docker compose build
```

Manual: open the app → **Terminal** tab → a shell prompt from the target host
appears, commands run, `Ctrl+C`/arrows/resize work, closing the SSH session shows
`Desconectado`.

---

## 9. Files touched

**Backend**

- `internal/config/config.go` (SSH config)
- `internal/terminal/model.go` (new)
- `internal/terminal/ssh.go` (new)
- `internal/terminal/session.go` (new)
- `internal/terminal/handler.go` (new)
- `cmd/api/main.go` (wiring)
- `go.mod` / `go.sum` (deps)

**Frontend**

- `package.json` (`@xterm/xterm`, `@xterm/addon-fit`)
- `src/services/terminalSocket.js` (new)
- `src/hooks/useTerminalSession.js` (new)
- `src/pages/TerminalPage/TerminalPage.jsx` (new)
- `src/pages/TerminalPage/TerminalPage.css` (new)
- `src/services/notifications.js`
- `src/App.jsx`
- `src/components/AppShell.jsx`
- `nginx.conf`
- `.env.example`

**Infra / docs**

- `docker-compose.yml` (SSH env)
- `README.md` (feature description)
