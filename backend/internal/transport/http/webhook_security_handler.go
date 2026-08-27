package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type WebhookSecurityHandler struct {
	service *service.WebhookSecurityService
}

func NewWebhookSecurityHandler(service *service.WebhookSecurityService) *WebhookSecurityHandler {
	return &WebhookSecurityHandler{service: service}
}

func (h *WebhookSecurityHandler) Save(w http.ResponseWriter, r *http.Request) {
	var input service.SaveWebhookSecurityInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid webhook security configuration")
		return
	}
	result, err := h.service.Save(r.Context(), chi.URLParam(r, "endpointId"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "WEBHOOK_SECURITY_CONFIG_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *WebhookSecurityHandler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Get(r.Context(), chi.URLParam(r, "endpointId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Webhook security configuration not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *WebhookSecurityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "endpointId")); err != nil {
		writeError(w, http.StatusBadRequest, "WEBHOOK_SECURITY_DELETE_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
