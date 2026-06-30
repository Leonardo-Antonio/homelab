package music

import (
	"errors"
	"net/http"
	"strconv"

	"homelab/backend/internal/httpapi"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/music/search", h.search)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Query parameter 'q' is required.", nil)
		return
	}

	limit := 12
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	results, err := h.service.Search(r.Context(), query, r.URL.Query().Get("type"), limit)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			httpapi.WriteError(w, http.StatusServiceUnavailable, "NotConfigured",
				"Configura SPOTIFY_CLIENT_ID y SPOTIFY_CLIENT_SECRET en el servidor para buscar en Spotify.", nil)
			return
		}
		httpapi.WriteError(w, http.StatusBadGateway, "UpstreamError", "No se pudo consultar Spotify.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}
