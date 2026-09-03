package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type TestSuiteHandler struct {
	suiteService *service.TestSuiteService
}

func NewTestSuiteHandler(suiteService *service.TestSuiteService) *TestSuiteHandler {
	return &TestSuiteHandler{suiteService: suiteService}
}

func (h *TestSuiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "Proje ID zorunludur")
		return
	}

	var params service.CreateTestSuiteParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek gövdesi")
		return
	}
	params.ProjectID = projectID

	suite, err := h.suiteService.CreateSuite(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, suite)
}

func (h *TestSuiteHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "Proje ID zorunludur")
		return
	}

	suites, err := h.suiteService.ListSuites(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"suites": suites,
		"count":  len(suites),
	})
}

func (h *TestSuiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	suiteID := chi.URLParam(r, "id")
	if suiteID == "" {
		writeError(w, http.StatusBadRequest, "SUITE_ID_REQUIRED", "Test paketi ID zorunludur")
		return
	}

	suite, runs, err := h.suiteService.GetSuite(r.Context(), suiteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"suite": suite,
		"runs":  runs,
	})
}

func (h *TestSuiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	suiteID := chi.URLParam(r, "id")
	projectID := r.URL.Query().Get("projectId")

	if suiteID == "" {
		writeError(w, http.StatusBadRequest, "SUITE_ID_REQUIRED", "Test paketi ID zorunludur")
		return
	}

	if err := h.suiteService.DeleteSuite(r.Context(), suiteID, projectID); err != nil {
		writeError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Test paketi silindi"})
}

func (h *TestSuiteHandler) Run(w http.ResponseWriter, r *http.Request) {
	suiteID := chi.URLParam(r, "id")
	if suiteID == "" {
		writeError(w, http.StatusBadRequest, "SUITE_ID_REQUIRED", "Test paketi ID zorunludur")
		return
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	clientIP := r.RemoteAddr

	report, err := h.suiteService.RunSuite(r.Context(), suiteID, userID, clientIP)
	if err != nil {
		writeError(w, http.StatusBadRequest, "RUN_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}
