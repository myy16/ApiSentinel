package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type AlertHandler struct {
	alertService *service.AlertService
}

func NewAlertHandler(alertService *service.AlertService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
	}
}

func (h *AlertHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/projects/{projectId}/alerts", h.CreateChannel)
	r.Get("/api/projects/{projectId}/alerts", h.ListChannels)
	r.Delete("/api/alerts/{id}", h.DeleteChannel)
	r.Post("/api/alerts/{id}/test", h.SendTestAlert)
}

func (h *AlertHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Project ID is required")
		return
	}

	var input service.CreateAlertChannelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}
	input.ProjectID = projectID

	channel, err := h.alertService.CreateChannel(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, channel)
}

func (h *AlertHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Project ID is required")
		return
	}

	channels, err := h.alertService.ListChannels(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, channels)
}

func (h *AlertHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Channel ID is required")
		return
	}

	if err := h.alertService.DeleteChannel(r.Context(), channelID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Alert channel deleted"})
}

func (h *AlertHandler) SendTestAlert(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Channel ID is required")
		return
	}

	if err := h.alertService.SendTestAlert(r.Context(), channelID); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Test alert dispatched successfully!"})
}
