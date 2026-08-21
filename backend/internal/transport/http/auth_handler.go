package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	OrganizationName string `json:"organizationName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Geçersiz istek gövdesi")
		return
	}

	if req.Email == "" || req.Password == "" || req.OrganizationName == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Tüm alanlar zorunludur")
		return
	}

	res, err := h.authService.Register(r.Context(), req.Email, req.Password, req.OrganizationName)
	if err != nil {
		writeError(w, http.StatusConflict, "EMAIL_EXISTS", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "Geçersiz istek gövdesi")
		return
	}

	res, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userId == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Yetkisiz erişim")
		return
	}

	user, orgs, err := h.authService.GetMe(r.Context(), userId)
	if err != nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "Kullanıcı bulunamadı")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":          user,
		"organizations": orgs,
	})
}
