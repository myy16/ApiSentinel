package http

import (
	"net/http"
	"strconv"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type RequestHandler struct {
	requestService *service.RequestService
}

func NewRequestHandler(requestService *service.RequestService) *RequestHandler {
	return &RequestHandler{requestService: requestService}
}

func (h *RequestHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	requests, err := h.requestService.ListByProject(r.Context(), projectId, int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"requests": requests,
	})
}
