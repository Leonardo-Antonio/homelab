package network

import (
	"database/sql"
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
	mux.HandleFunc("GET /api/v1/network/overview", h.overview)
	mux.HandleFunc("GET /api/v1/network/devices", h.devices)
	mux.HandleFunc("GET /api/v1/network/visits", h.visits)
	mux.HandleFunc("GET /api/v1/network/snapshot", h.snapshot)
	mux.HandleFunc("PATCH /api/v1/network/devices/{id}", h.updateDevice)
}

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not load network overview.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, snapshot.Overview)
}

func (h *Handler) devices(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not load network devices.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"items": snapshot.Devices})
}

func (h *Handler) visits(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not load network visits.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"items": snapshot.Visits})
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not load network snapshot.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, snapshot)
}

func (h *Handler) updateDevice(w http.ResponseWriter, r *http.Request) {
	var request UpdateDeviceRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Invalid device update payload.", nil)
		return
	}
	device, err := h.service.UpdateDevice(r.Context(), r.PathValue("id"), request)
	if errors.Is(err, ErrInvalidDeviceStatus) {
		httpapi.WriteError(w, http.StatusBadRequest, "BadRequest", "Invalid device status.", map[string]string{"status": request.Status})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		httpapi.WriteError(w, http.StatusNotFound, "NotFound", "Network device was not found.", nil)
		return
	}
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "InternalServerError", "Could not update network device.", nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, device)
}
