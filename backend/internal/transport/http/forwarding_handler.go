package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type ForwardingHandler struct {
	fwdService *service.ForwardingService
}

func NewForwardingHandler(fwdService *service.ForwardingService) *ForwardingHandler {
	return &ForwardingHandler{
		fwdService: fwdService,
	}
}

func (h *ForwardingHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/endpoints/{endpointId}/forwarding", h.SaveConfig)
	r.Get("/api/endpoints/{endpointId}/forwarding", h.GetConfig)
	r.Get("/api/endpoints/{endpointId}/dlq", h.ListDLQ)
}

func (h *ForwardingHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	endpointID := chi.URLParam(r, "endpointId")
	if endpointID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Endpoint ID is required")
		return
	}

	var input service.SaveForwardingConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}
	input.EndpointID = endpointID

	cfg, err := h.fwdService.SaveConfig(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (h *ForwardingHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	endpointID := chi.URLParam(r, "endpointId")
	if endpointID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Endpoint ID is required")
		return
	}

	cfg, err := h.fwdService.GetConfig(r.Context(), endpointID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Forwarding configuration not found")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (h *ForwardingHandler) ListDLQ(w http.ResponseWriter, r *http.Request) {
	endpointID := chi.URLParam(r, "endpointId")
	if endpointID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Endpoint ID is required")
		return
	}

	dlqRecords, err := h.fwdService.ListDLQ(r.Context(), endpointID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dlqRecords)
}
