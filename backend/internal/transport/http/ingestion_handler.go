package http

import (
	"io"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type IngestionHandler struct {
	ingestionService *service.IngestionService
}

func NewIngestionHandler(ingestionService *service.IngestionService) *IngestionHandler {
	return &IngestionHandler{ingestionService: ingestionService}
}

func (h *IngestionHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "SLUG_REQUIRED", "Slug belirtilmedi")
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	result, err := h.ingestionService.ProcessWebhook(
		r.Context(),
		slug,
		r.Method,
		r.Header,
		r.URL.Query(),
		bodyBytes,
		clientIP,
	)

	if err != nil {
		writeError(w, http.StatusNotFound, "ENDPOINT_NOT_FOUND", err.Error())
		return
	}

	writeJSON(w, result.StatusCode, result.ResponseBody)
}
