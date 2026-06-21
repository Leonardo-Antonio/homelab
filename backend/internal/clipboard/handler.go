package clipboard

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"homelab/backend/internal/httpapi"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/clipboard-items", h.list)
	mux.HandleFunc("POST /api/v1/clipboard-items", h.create)
	mux.HandleFunc("DELETE /api/v1/clipboard-items", h.deleteAll)
	mux.HandleFunc("GET /api/v1/clipboard-items/{id}", h.get)
	mux.HandleFunc("DELETE /api/v1/clipboard-items/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "pageSize", DefaultPageSize)

	response, err := h.service.List(r.Context(), page, pageSize)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not list clipboard items.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Clipboard item was not found.", nil)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not get clipboard item.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request CreateItemRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request body must be valid JSON.", nil)
		return
	}

	item, err := h.service.Create(r.Context(), request.Text)
	if errors.Is(err, ErrInvalidText) {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError", "Text is required and must be shorter than 20000 characters.", map[string]string{
			"text": "required",
		})
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not create clipboard item.", nil)
		return
	}

	w.Header().Set("Location", "/api/v1/clipboard-items/"+item.ID)
	httpapi.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Clipboard item was not found.", nil)
		return
	} else if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not delete clipboard item.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) deleteAll(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteAll(r.Context()); err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not delete clipboard items.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusNoContent, nil)
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
