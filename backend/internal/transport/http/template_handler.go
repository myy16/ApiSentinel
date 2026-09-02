package http

import (
	"net/http"

	"github.com/apisentinel/apisentinel/internal/security/hmac"
)

type TemplateHandler struct{}

func NewTemplateHandler() *TemplateHandler {
	return &TemplateHandler{}
}

// ListProviders returns all available pre-configured webhook provider templates.
func (h *TemplateHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	templates := hmac.GetAvailableTemplates()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers": templates,
		"count":     len(templates),
	})
}
