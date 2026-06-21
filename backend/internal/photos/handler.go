package photos

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
	mux.HandleFunc("GET /api/v1/photos", h.list)
	mux.HandleFunc("POST /api/v1/photos", h.create)
	mux.HandleFunc("GET /api/v1/photos/{id}", h.get)
	mux.HandleFunc("GET /api/v1/photos/{id}/file", h.file)
	mux.HandleFunc("DELETE /api/v1/photos/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	response, err := h.service.List(
		r.Context(),
		parseQueryInt(r, "page", 1),
		parseQueryInt(r, "pageSize", DefaultPageSize),
	)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not list photos.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	photo, err := h.service.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Photo was not found.", nil)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not get photo.", nil)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, photo)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+1024)
	if err := r.ParseMultipartForm(MaxUploadBytes + 1024); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request must include a valid multipart photo.", nil)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Photo file is required.", nil)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	photo, err := h.service.Create(r.Context(), file, contentType)
	if errors.Is(err, ErrInvalidPhoto) {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError", "Photo must be a valid jpeg or png image up to 8MB.", nil)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not save photo.", nil)
		return
	}

	w.Header().Set("Location", "/api/v1/photos/"+photo.ID)
	httpapi.WriteJSON(w, http.StatusCreated, photo)
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	photo, err := h.service.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Photo was not found.", nil)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not get photo.", nil)
		return
	}

	w.Header().Set("Content-Type", photo.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, h.service.FilePath(photo))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); errors.Is(err, ErrNotFound) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Photo was not found.", nil)
		return
	} else if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not delete photo.", nil)
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
