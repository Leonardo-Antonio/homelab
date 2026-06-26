package notes

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
	mux.HandleFunc("GET /api/v1/notes", h.list)
	mux.HandleFunc("POST /api/v1/notes", h.create)
	mux.HandleFunc("GET /api/v1/notes/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/notes/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.service.List(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not list notes.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, nodes)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Note not found.", nil)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not get note.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request body must be valid JSON.", nil)
		return
	}

	node, err := h.service.Create(r.Context(), req)
	switch {
	case errors.Is(err, ErrInvalidName):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Name is required and must be shorter than 255 characters.",
			map[string]string{"name": "required"})
	case errors.Is(err, ErrInvalidType):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Type must be 'dir' or 'note'.",
			map[string]string{"type": "invalid"})
	case errors.Is(err, ErrContentTooLong):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Content exceeds the maximum allowed length.",
			map[string]string{"content": "too_long"})
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not create note.", nil)
	default:
		w.Header().Set("Location", "/api/v1/notes/"+node.ID)
		httpapi.WriteJSON(w, http.StatusCreated, node)
	}
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request body must be valid JSON.", nil)
		return
	}

	node, err := h.service.Update(r.Context(), r.PathValue("id"), req)
	switch {
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Note not found.", nil)
	case errors.Is(err, ErrInvalidName):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Name is required and must be shorter than 255 characters.",
			map[string]string{"name": "required"})
	case errors.Is(err, ErrContentTooLong):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Content exceeds the maximum allowed length.",
			map[string]string{"content": "too_long"})
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not update note.", nil)
	default:
		httpapi.WriteJSON(w, http.StatusOK, node)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	err := h.service.Delete(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Note not found.", nil)
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not delete note.", nil)
	default:
		httpapi.WriteJSON(w, http.StatusNoContent, nil)
	}
}
