package terminal

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"homelab/backend/internal/config"
	"homelab/backend/internal/httpapi"

	"github.com/coder/websocket"
)

type Handler struct {
	cfg           config.SSHConfig
	allowedOrigin string
}

func NewHandler(cfg config.SSHConfig, allowedOrigin string) *Handler {
	return &Handler{cfg: cfg, allowedOrigin: allowedOrigin}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/terminal/info", h.info)
	mux.HandleFunc("GET /api/v1/terminal/ws", h.connect)
}

func (h *Handler) info(w http.ResponseWriter, _ *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, InfoResponse{
		Enabled: h.cfg.Enabled,
		Host:    h.cfg.Host,
		Port:    h.cfg.Port,
		User:    h.cfg.User,
	})
}

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled {
		httpapi.WriteError(w, http.StatusServiceUnavailable, "TerminalDisabled", "The SSH terminal is disabled.", nil)
		return
	}

	// The server's Read/Write timeouts set deadlines on the connection that
	// would abort this long-lived session (coder/websocket uses context-based
	// timeouts and does not reset them). Clear them before the upgrade.
	responseController := http.NewResponseController(w)
	_ = responseController.SetReadDeadline(time.Time{})
	_ = responseController.SetWriteDeadline(time.Time{})

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.originPatterns(),
	})
	if err != nil {
		// websocket.Accept already wrote an HTTP error response.
		slog.Warn("terminal websocket accept failed", "error", err)
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(maxMessageBytes)

	// The session lifetime is bound to the request context, not the server
	// write timeout: coder/websocket owns the hijacked connection.
	ctx := r.Context()

	client, err := dial(ctx, h.cfg)
	if err != nil {
		slog.Warn("terminal ssh dial failed", "error", err)
		writeStatus(ctx, conn, statusMessage{Type: "error", Message: "No se pudo conectar por SSH."})
		conn.Close(websocket.StatusInternalError, "ssh connection failed")
		return
	}
	defer client.Close()

	writeStatus(ctx, conn, statusMessage{Type: "status", State: "connected"})

	if err := runSession(ctx, conn, client); err != nil {
		slog.Warn("terminal session ended with error", "error", err)
		conn.Close(websocket.StatusInternalError, "session error")
		return
	}

	conn.Close(websocket.StatusNormalClosure, "session closed")
}

// originPatterns translates ALLOWED_ORIGIN into host patterns for the WebSocket
// origin check. "*" allows any origin; otherwise each origin's host is allowed.
func (h *Handler) originPatterns() []string {
	if strings.TrimSpace(h.allowedOrigin) == "*" {
		return []string{"*"}
	}

	var patterns []string
	for _, origin := range strings.Split(h.allowedOrigin, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}

		host := origin
		if index := strings.Index(host, "://"); index != -1 {
			host = host[index+3:]
		}
		patterns = append(patterns, host)
	}

	return patterns
}
