package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type EndpointHandler struct {
	endpointService *service.EndpointService
}

func NewEndpointHandler(endpointService *service.EndpointService) *EndpointHandler {
	return &EndpointHandler{endpointService: endpointService}
}

type CreateEndpointRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Mode        string  `json:"mode"`
	UpstreamURL *string `json:"upstreamUrl"`
}

func (h *EndpointHandler) List(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	endpoints, err := h.endpointService.ListEndpoints(r.Context(), projectId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"endpoints": endpoints,
	})
}

func (h *EndpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")

	var req CreateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Endpoint adı zorunludur")
		return
	}

	ep, err := h.endpointService.CreateEndpoint(r.Context(), projectId, req.Name, req.Slug, req.Mode, req.UpstreamURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "SLUG_EXISTS", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ep)
}
