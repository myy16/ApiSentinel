package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/google/uuid"
)

type AISettingsHandler struct {
	aiSettingsService *service.AISettingsService
}

func NewAISettingsHandler(aiSettingsService *service.AISettingsService) *AISettingsHandler {
	return &AISettingsHandler{aiSettingsService: aiSettingsService}
}

func getOrgIDFromContext(r *http.Request) string {
	pgOrgID := middleware.GetOrganizationID(r.Context())
	if pgOrgID.Valid {
		return uuid.UUID(pgOrgID.Bytes).String()
	}
	if val, ok := r.Context().Value(middleware.OrgIDKey).(string); ok && val != "" {
		return val
	}
	return ""
}

func (h *AISettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromContext(r)
	if orgID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Organizasyon kimliği bulunamadı")
		return
	}

	settings, err := h.aiSettingsService.GetSettings(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

type UpdateAISettingsRequest struct {
	AIEnabled           bool     `json:"aiEnabled"`
	AIDataSharingLevel  string   `json:"aiDataSharingLevel"`
	CustomRedactionKeys []string `json:"customRedactionKeys"`
}

func (h *AISettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	orgID := getOrgIDFromContext(r)
	if orgID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Organizasyon kimliği bulunamadı")
		return
	}

	var req UpdateAISettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Geçersiz istek gövdesi")
		return
	}

	settings, err := h.aiSettingsService.UpdateSettings(r.Context(), service.UpdateAISettingsInput{
		OrganizationID:      orgID,
		AIEnabled:           req.AIEnabled,
		AIDataSharingLevel:  req.AIDataSharingLevel,
		CustomRedactionKeys: req.CustomRedactionKeys,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

type TestSanitizeRequest struct {
	SampleText       string   `json:"sampleText"`
	CustomRedactKeys []string `json:"customRedactKeys"`
}

func (h *AISettingsHandler) TestSanitize(w http.ResponseWriter, r *http.Request) {
	var req TestSanitizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SampleText == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Test edilecek metin (sampleText) zorunludur")
		return
	}

	res := h.aiSettingsService.TestSanitization(service.TestSanitizeInput{
		SampleText:       req.SampleText,
		CustomRedactKeys: req.CustomRedactKeys,
	})

	writeJSON(w, http.StatusOK, res)
}
