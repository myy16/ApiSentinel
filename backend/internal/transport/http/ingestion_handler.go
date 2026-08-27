package http

import (
	"io"
	"net"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

const maxWebhookBodyBytes int64 = 5 << 20 // 5 MiB

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

	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Webhook payload exceeds the 5 MiB limit")
		return
	}

	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
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
