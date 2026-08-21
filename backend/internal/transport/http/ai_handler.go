package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/ai"
)

type AIHandler struct {
	explainer *ai.Explainer
}

func NewAIHandler(explainer *ai.Explainer) *AIHandler {
	return &AIHandler{explainer: explainer}
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
