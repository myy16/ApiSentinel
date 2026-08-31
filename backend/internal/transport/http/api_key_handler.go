package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type APIKeyHandler struct {
	keyService *service.APIKeyService
}

func NewAPIKeyHandler(keyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{keyService: keyService}
}

type CreateKeyRequest struct {
	Name      string     `json:"name"`
	IsLive    bool       `json:"isLive"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Project ID required")
		return
	}

	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "KEY_NAME_REQUIRED", "API key name is required")
		return
	}
	if len([]rune(req.Name)) > 100 {
		writeError(w, http.StatusBadRequest, "KEY_NAME_TOO_LONG", "API key name must be 100 characters or fewer")
		return
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	created, err := h.keyService.GenerateAPIKey(r.Context(), projectID, req.Name, userID, req.IsLive, req.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "KEY_CREATION_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"apiKey":  created,
	})
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Project ID required")
		return
	}

	keys, err := h.keyService.ListByProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keys": keys,
	})
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	keyID := chi.URLParam(r, "keyId")

	if projectID == "" || keyID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Project ID and Key ID required")
		return
	}

	if err := h.keyService.RevokeKey(r.Context(), projectID, keyID); err != nil {
		writeError(w, http.StatusInternalServerError, "REVOKE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "API key revoked successfully",
	})
}
