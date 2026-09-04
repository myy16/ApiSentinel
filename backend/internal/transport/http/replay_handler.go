package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/apisentinel/apisentinel/internal/middleware"
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
	TargetURL           string            `json:"targetUrl"`
	Environment         string            `json:"environment"`
	CustomHeaders       map[string]string `json:"customHeaders,omitempty"`
	Justification       string            `json:"justification,omitempty"`
	OverrideIdempotency bool              `json:"overrideIdempotency"`
	RenewIdempotency    bool              `json:"renewIdempotency"`
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

	// Enforce Owner Authorization & Justification for Idempotency Override
	if req.OverrideIdempotency {
		userRole := middleware.GetRole(r.Context())
		if userRole != "OWNER" {
			writeError(w, http.StatusForbidden, "ROLE_UNAUTHORIZED", "Yalnızca organizasyon OWNER rolü idempotency override ile replay yapabilir")
			return
		}
		if req.Justification == "" {
			writeError(w, http.StatusBadRequest, "JUSTIFICATION_REQUIRED", "Idempotency override edilirken bir gerekçe (justification) belirtilmesi zorunludur")
			return
		}
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	clientIP := r.RemoteAddr

	result, err := h.replayService.ExecuteReplay(r.Context(), service.ExecuteReplayParams{
		SourceRequestId:     requestId,
		TargetURL:           req.TargetURL,
		Environment:         req.Environment,
		CustomHeaders:       req.CustomHeaders,
		Justification:       req.Justification,
		OverrideIdempotency: req.OverrideIdempotency,
		RenewIdempotency:    req.RenewIdempotency,
		UserID:              userID,
		ClientIP:            clientIP,
	})
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
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
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

func (h *ReplayHandler) GetReplay(w http.ResponseWriter, r *http.Request) {
	replayId := chi.URLParam(r, "id")
	if replayId == "" {
		writeError(w, http.StatusBadRequest, "REPLAY_ID_REQUIRED", "Replay ID belirtilmedi")
		return
	}

	job, err := h.replayService.GetReplayJob(r.Context(), replayId)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job)
}
