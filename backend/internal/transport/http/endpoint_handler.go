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
	Name                string  `json:"name"`
	Slug                string  `json:"slug"`
	Mode                string  `json:"mode"`
	UpstreamURL         *string `json:"upstreamUrl"`
	MaxPayloadSizeBytes int32   `json:"maxPayloadSizeBytes"`
	RateLimitRpm        int32   `json:"rateLimitRpm"`
	BurstThreshold      int32   `json:"burstThreshold"`
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

	ep, err := h.endpointService.CreateEndpoint(r.Context(), service.CreateEndpointInput{
		ProjectID:           projectId,
		Name:                req.Name,
		Slug:                req.Slug,
		Mode:                req.Mode,
		UpstreamURL:         req.UpstreamURL,
		MaxPayloadSizeBytes: req.MaxPayloadSizeBytes,
		RateLimitRpm:        req.RateLimitRpm,
		BurstThreshold:      req.BurstThreshold,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "SLUG_EXISTS", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ep)
}

type UpdateEndpointRequest struct {
	Name                string  `json:"name"`
	Mode                string  `json:"mode"`
	IsActive            *bool   `json:"isActive"`
	UpstreamURL         *string `json:"upstreamUrl"`
	MaxPayloadSizeBytes *int32  `json:"maxPayloadSizeBytes"`
	RateLimitRpm        *int32  `json:"rateLimitRpm"`
	BurstThreshold      *int32  `json:"burstThreshold"`
}

func (h *EndpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	endpointId := chi.URLParam(r, "endpointId")

	var req UpdateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Geçersiz istek gövdesi")
		return
	}

	ep, err := h.endpointService.UpdateEndpoint(r.Context(), service.UpdateEndpointInput{
		EndpointID:          endpointId,
		ProjectID:           projectId,
		Name:                req.Name,
		Mode:                req.Mode,
		IsActive:            req.IsActive,
		UpstreamURL:         req.UpstreamURL,
		MaxPayloadSizeBytes: req.MaxPayloadSizeBytes,
		RateLimitRpm:        req.RateLimitRpm,
		BurstThreshold:      req.BurstThreshold,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ep)
}

func (h *EndpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	endpointId := chi.URLParam(r, "endpointId")

	if err := h.endpointService.DeleteEndpoint(r.Context(), endpointId, projectId); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Endpoint başarıyla silindi"})
}


func (h *EndpointHandler) SaveSchema(w http.ResponseWriter, r *http.Request) {
	endpointId := chi.URLParam(r, "endpointId")

	var schemaData json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&schemaData); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SCHEMA", "Geçersiz JSON Schema verisi")
		return
	}

	res, err := h.endpointService.SaveSchema(r.Context(), endpointId, []byte(schemaData))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *EndpointHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
	endpointId := chi.URLParam(r, "endpointId")

	res, err := h.endpointService.GetSchema(r.Context(), endpointId)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Bu endpoint için tanımlı JSON Schema bulunamadı")
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *EndpointHandler) DeleteSchema(w http.ResponseWriter, r *http.Request) {
	endpointId := chi.URLParam(r, "endpointId")

	if err := h.endpointService.DeleteSchema(r.Context(), endpointId); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "JSON Schema sözleşmesi silindi"})
}

