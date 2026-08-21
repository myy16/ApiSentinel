package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type MockHandler struct {
	mockService *service.MockService
}

func NewMockHandler(mockService *service.MockService) *MockHandler {
	return &MockHandler{mockService: mockService}
}

func (h *MockHandler) Create(w http.ResponseWriter, r *http.Request) {
	endpointId := chi.URLParam(r, "endpointId")

	var req service.CreateMockRuleInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Kural adı zorunludur")
		return
	}

	rule, err := h.mockService.CreateRule(r.Context(), endpointId, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

func (h *MockHandler) List(w http.ResponseWriter, r *http.Request) {
	endpointId := chi.URLParam(r, "endpointId")

	rules, err := h.mockService.ListRules(r.Context(), endpointId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mocks": rules,
	})
}
