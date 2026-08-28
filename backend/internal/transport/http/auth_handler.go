package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"unicode"

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

	if !isValidEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Geçersiz e-posta formatı")
		return
	}

	if err := validatePassword(req.Password); err != "" {
		writeError(w, http.StatusBadRequest, "WEAK_PASSWORD", err)
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

// Logout provides a stateless logout endpoint. No server-side invalidation;
// the frontend clears local tokens. This prevents the frontend from hitting a 404 (#18).
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Çıkış başarılı",
	})
}

// Refresh validates a refresh token and issues a new access token (#17).
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "refreshToken alanı zorunludur")
		return
	}

	res, err := h.authService.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// validatePassword enforces minimum password complexity.
// Returns empty string if valid, or the validation error message.
func validatePassword(password string) string {
	if len(password) < 8 {
		return "Şifre en az 8 karakter olmalıdır"
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper {
		return "Şifre en az bir büyük harf içermelidir"
	}
	if !hasLower {
		return "Şifre en az bir küçük harf içermelidir"
	}
	if !hasDigit {
		return "Şifre en az bir rakam içermelidir"
	}
	return ""
}
