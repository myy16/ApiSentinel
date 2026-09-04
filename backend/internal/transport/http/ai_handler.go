package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/ai"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/middleware"
)

type AIHandler struct {
	explainer *ai.Explainer
	queries   *database.Queries
}

func NewAIHandler(explainer *ai.Explainer, queries ...*database.Queries) *AIHandler {
	var q *database.Queries
	if len(queries) > 0 {
		q = queries[0]
	}
	return &AIHandler{explainer: explainer, queries: q}
}

type ExplainFindingRequest struct {
	Category       string `json:"category"`
	FindingType    string `json:"findingType"`
	Severity       string `json:"severity"`
	MaskedEvidence string `json:"maskedEvidence"`
	Message        string `json:"message"`
}

func (h *AIHandler) ExplainFinding(w http.ResponseWriter, r *http.Request) {
	var req ExplainFindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Geçersiz istek gövdesi")
		return
	}

	// Enforce Organization AI Opt-in Policy
	if h.queries != nil {
		orgID := middleware.GetOrganizationID(r.Context())
		if orgID.Valid {
			if orgSettings, orgErr := h.queries.GetOrganizationAISettings(r.Context(), orgID); orgErr == nil {
				if !orgSettings.AiEnabled {
					writeError(w, http.StatusForbidden, "AI_DISABLED", "Bu organizasyon için AI analizi devre dışı bırakılmıştır")
					return
				}
			}
		}
	}

	explanation, err := h.explainer.ExplainFinding(
		r.Context(),
		req.Category,
		req.FindingType,
		req.Severity,
		req.MaskedEvidence,
		req.Message,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, explanation)
}
