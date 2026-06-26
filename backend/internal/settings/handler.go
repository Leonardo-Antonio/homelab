package settings

import (
	"errors"
	"net/http"

	"homelab/backend/internal/httpapi"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings", h.get)
	mux.HandleFunc("PUT /api/v1/settings", h.update)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	current, err := h.service.Get(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not load settings.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, current)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req Settings
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request body must be valid JSON.", nil)
		return
	}

	updated, err := h.service.Update(r.Context(), req)
	switch {
	case errors.Is(err, ErrInvalidSettings):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Theme, language or font has an unsupported value.", nil)
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not save settings.", nil)
	default:
		httpapi.WriteJSON(w, http.StatusOK, updated)
	}
}
