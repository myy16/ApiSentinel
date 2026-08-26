package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type ProjectHandler struct {
	projectService *service.ProjectService
}

func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	orgId := r.Context().Value(middleware.OrgIDKey).(string)
	projects, err := h.projectService.ListProjects(r.Context(), orgId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"projects": projects,
	})
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgId := r.Context().Value(middleware.OrgIDKey).(string)

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Proje adı zorunludur")
		return
	}

	p, err := h.projectService.CreateProject(r.Context(), orgId, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgId := r.Context().Value(middleware.OrgIDKey).(string)
	projectId := chi.URLParam(r, "id")

	p, err := h.projectService.GetProject(r.Context(), orgId, projectId)
	if err != nil {
		writeError(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "Proje bulunamadı")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgId := r.Context().Value(middleware.OrgIDKey).(string)
	projectId := chi.URLParam(r, "id")

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Proje adı zorunludur")
		return
	}

	p, err := h.projectService.UpdateProject(r.Context(), orgId, projectId, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgId := r.Context().Value(middleware.OrgIDKey).(string)
	projectId := chi.URLParam(r, "id")

	if err := h.projectService.DeleteProject(r.Context(), orgId, projectId); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
