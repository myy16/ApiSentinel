package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
)

type ReplayHandler struct {
	replayService *service.ReplayService
}

func NewReplayHandler(replayService *service.ReplayService) *ReplayHandler {
	return &ReplayHandler{replayService: replayService}
}

type ExecuteReplayRequest struct {
	TargetURL string `json:"targetUrl"`
}

func (h *ReplayHandler) Execute(w http.ResponseWriter, r *http.Request) {
	requestId := chi.URLParam(r, "id")
	if requestId == "" {
		writeError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "İstek ID belirtilmedi")
		return
	}

	var req ExecuteReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetURL == "" {
		writeError(w, http.StatusBadRequest, "TARGET_URL_REQUIRED", "Hedef URL (targetUrl) zorunludur")
		return
	}

	result, err := h.replayService.ExecuteReplay(r.Context(), requestId, req.TargetURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "REPLAY_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ReplayHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	jobs, err := h.replayService.ListReplayJobs(r.Context(), projectId, int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"replays": jobs,
	})
}
