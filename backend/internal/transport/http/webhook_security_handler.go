package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type WebhookSecurityHandler struct {
	service *service.WebhookSecurityService
	queries *database.Queries
}

func NewWebhookSecurityHandler(service *service.WebhookSecurityService, queries ...*database.Queries) *WebhookSecurityHandler {
	var q *database.Queries
	if len(queries) > 0 {
		q = queries[0]
	}
	return &WebhookSecurityHandler{service: service, queries: q}
}

func (h *WebhookSecurityHandler) Save(w http.ResponseWriter, r *http.Request) {
	endpointIdStr := chi.URLParam(r, "endpointId")
	var input service.SaveWebhookSecurityInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid webhook security configuration")
		return
	}
	result, err := h.service.Save(r.Context(), endpointIdStr, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "WEBHOOK_SECURITY_CONFIG_FAILED", err.Error())
		return
	}

	// Write Audit Log for SECRET_UPDATED
	if h.queries != nil {
		orgID := middleware.GetOrganizationID(r.Context())
		userID := middleware.GetUserID(r.Context())
		epUUID, _ := uuid.Parse(endpointIdStr)

		var projUUID pgtype.UUID
		if ep, err := h.queries.GetEndpointByIDOnly(r.Context(), pgtype.UUID{Bytes: epUUID, Valid: true}); err == nil {
			projUUID = ep.ProjectID
		}

		metaJSON, _ := json.Marshal(map[string]interface{}{
			"endpointId": endpointIdStr,
			"provider":   input.Provider,
		})

		_, auditErr := h.queries.CreateAuditLog(r.Context(), database.CreateAuditLogParams{
			OrganizationID: orgID,
			ProjectID:      projUUID,
			UserID:         userID,
			Action:         "SECRET_UPDATED",
			ResourceType:   "WEBHOOK_SECURITY",
			ResourceID:     endpointIdStr,
			Justification:  pgtype.Text{String: "Webhook HMAC gizli anahtarı güncellendi", Valid: true},
			IpAddress:      pgtype.Text{String: r.RemoteAddr, Valid: true},
			Metadata:       metaJSON,
		})
		if auditErr != nil {
			log.Error().Err(auditErr).Msg("Failed to write audit log for SECRET_UPDATED")
		}
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
