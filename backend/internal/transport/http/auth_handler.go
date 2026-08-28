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

	setAuthCookies(w, res.AccessToken, res.RefreshToken)
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

	setAuthCookies(w, res.AccessToken, res.RefreshToken)
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

// Logout clears the auth cookies and returns 200 (#18, #19).
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Çıkış başarılı",
	})
}

// Refresh validates a refresh token (from body or cookie) and issues a new access token (#17, #19).
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookie, err := r.Cookie("apisentinel_refresh_token"); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "refreshToken alanı veya cookie'si zorunludur")
		return
	}

	res, err := h.authService.RefreshAccessToken(r.Context(), refreshToken)
	if err != nil {
		clearAuthCookies(w)
		writeError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", err.Error())
		return
	}

	setAccessTokenCookie(w, res.AccessToken)
	writeJSON(w, http.StatusOK, res)
}

func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	setAccessTokenCookie(w, accessToken)
	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "apisentinel_refresh_token",
			Value:    refreshToken,
			Path:     "/api/auth",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   7 * 24 * 3600, // 7 days
		})
	}
}

func setAccessTokenCookie(w http.ResponseWriter, accessToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "apisentinel_access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 3600, // 24 hours
	})
}

func clearAuthCookies(w http.ResponseWriter, ) {
	http.SetCookie(w, &http.Cookie{
		Name:     "apisentinel_access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "apisentinel_refresh_token",
		Value:    "",
		Path:     "/api/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
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
