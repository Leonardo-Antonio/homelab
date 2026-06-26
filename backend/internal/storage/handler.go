package storage

import (
	"errors"
	"net/http"
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
	mux.HandleFunc("GET /api/v1/storage/nodes", h.list)
	mux.HandleFunc("GET /api/v1/storage/nodes/{id}", h.get)
	mux.HandleFunc("POST /api/v1/storage/folders", h.createFolder)
	mux.HandleFunc("POST /api/v1/storage/files", h.uploadFile)
	mux.HandleFunc("GET /api/v1/storage/files/{id}/content", h.download)
	mux.HandleFunc("PATCH /api/v1/storage/nodes/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/storage/nodes/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	parentID := optionalQuery(r, "parentId")
	response, err := h.service.List(r.Context(), parentID)
	switch {
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Folder not found.", nil)
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not list folder.", nil)
	default:
		httpapi.WriteJSON(w, http.StatusOK, response)
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	node, err := h.service.Get(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Node not found.", nil)
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not get node.", nil)
	default:
		httpapi.WriteJSON(w, http.StatusOK, node)
	}
}

func (h *Handler) createFolder(w http.ResponseWriter, r *http.Request) {
	var req CreateFolderRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request body must be valid JSON.", nil)
		return
	}

	node, err := h.service.CreateFolder(r.Context(), req)
	h.writeMutationResult(w, r, node, err, http.StatusCreated)
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+1<<20)

	// Stream the multipart body rather than buffering it to disk/memory: pull
	// the first "file" part and hand its reader straight to the service.
	reader, err := r.MultipartReader()
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request must be multipart/form-data.", nil)
		return
	}

	var parentID *string
	for {
		part, err := reader.NextPart()
		if err != nil {
			httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "A file part is required.", nil)
			return
		}

		switch part.FormName() {
		case "parentId":
			value := readSmallPart(part)
			part.Close()
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				parentID = &trimmed
			}
		case "file":
			fileName := part.FileName()
			contentType := part.Header.Get("Content-Type")
			node, createErr := h.service.CreateFile(r.Context(), parentID, fileName, contentType, part)
			part.Close()
			h.writeMutationResult(w, r, node, createErr, http.StatusCreated)
			return
		default:
			part.Close()
		}
	}
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	node, file, err := h.service.OpenFile(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound) || errors.Is(err, ErrNotAFile):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "File not found.", nil)
		return
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not open file.", nil)
		return
	}
	defer file.Close()

	if node.ContentType != "" {
		w.Header().Set("Content-Type", node.ContentType)
	}
	w.Header().Set("Content-Disposition", contentDisposition(r, node.Name))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	// ServeContent handles Range, If-Modified-Since and Content-Length.
	http.ServeContent(w, r, node.Name, node.UpdatedAt, file)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Request body must be valid JSON.", nil)
		return
	}

	node, err := h.service.Update(r.Context(), r.PathValue("id"), req)
	h.writeMutationResult(w, r, node, err, http.StatusOK)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	err := h.service.Delete(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Node not found.", nil)
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not delete node.", nil)
	default:
		httpapi.WriteJSON(w, http.StatusNoContent, nil)
	}
}

// writeMutationResult maps the shared set of create/update domain errors to
// HTTP responses so each handler stays small.
func (h *Handler) writeMutationResult(w http.ResponseWriter, _ *http.Request, node Node, err error, successStatus int) {
	switch {
	case errors.Is(err, ErrInvalidName):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"Name is required and must be shorter than 255 characters.",
			map[string]string{"name": "invalid"})
	case errors.Is(err, ErrInvalidParent):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"The target folder does not exist.",
			map[string]string{"parentId": "invalid"})
	case errors.Is(err, ErrNameConflict):
		httpapi.WriteError(w, http.StatusConflict, "Conflict",
			"A file or folder with that name already exists here.",
			map[string]string{"name": "conflict"})
	case errors.Is(err, ErrMoveIntoSelf):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError",
			"A folder cannot be moved into itself.",
			map[string]string{"parentId": "invalid"})
	case errors.Is(err, ErrEmptyUpload):
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "ValidationError", "The uploaded file is empty.", nil)
	case errors.Is(err, ErrUploadTooLarge):
		httpapi.WriteError(w, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "The uploaded file is too large.", nil)
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Node not found.", nil)
	case err != nil:
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not complete the request.", nil)
	default:
		if successStatus == http.StatusCreated {
			w.Header().Set("Location", "/api/v1/storage/nodes/"+node.ID)
		}
		httpapi.WriteJSON(w, successStatus, node)
	}
}

func optionalQuery(r *http.Request, key string) *string {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	return &value
}

// readSmallPart reads a bounded amount from a non-file form field.
func readSmallPart(part interface{ Read([]byte) (int, error) }) string {
	buf := make([]byte, 0, 256)
	tmp := make([]byte, 256)
	for len(buf) < 4096 {
		n, err := part.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return string(buf)
}

func contentDisposition(r *http.Request, name string) string {
	disposition := "inline"
	if strings.TrimSpace(r.URL.Query().Get("download")) != "" {
		disposition = "attachment"
	}
	// Escape quotes to keep the header well-formed.
	safe := strings.ReplaceAll(name, `"`, `\"`)
	return disposition + `; filename="` + safe + `"`
}
