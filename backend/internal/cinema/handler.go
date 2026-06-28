package cinema

import (
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
	mux.HandleFunc("GET /api/v1/cinema/sources", h.sources)
	mux.HandleFunc("GET /api/v1/cinema/search", h.search)
}

func (h *Handler) sources(w http.ResponseWriter, r *http.Request) {
	sources := h.service.Sources()
	items := make([]map[string]string, 0, len(sources))
	for _, source := range sources {
		items = append(items, map[string]string{"id": source.ID, "label": source.Label})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
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

	results := h.service.Search(r.Context(), query, r.URL.Query().Get("source"), limit)
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}
