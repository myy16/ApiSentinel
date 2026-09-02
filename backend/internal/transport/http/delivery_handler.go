package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/delivery"
	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type DeliveryHandler struct {
	queries     *database.Queries
	deliverySvc *service.DeliveryService
}

func NewDeliveryHandler(queries *database.Queries, deliverySvc *service.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{
		queries:     queries,
		deliverySvc: deliverySvc,
	}
}

// ListByEndpoint returns delivery jobs for an endpoint.
func (h *DeliveryHandler) ListByEndpoint(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := chi.URLParam(r, "endpointId")
	epUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint UUID")
		return
	}

	limit := int32(50)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	jobs, err := h.queries.ListDeliveryJobsByEndpoint(r.Context(), database.ListDeliveryJobsByEndpointParams{
		EndpointID: pgtype.UUID{Bytes: epUUID, Valid: true},
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list delivery jobs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deliveries": jobs,
		"count":      len(jobs),
	})
}

// GetTimeline returns the complete lifecycle timeline and all attempt records for a delivery job.
func (h *DeliveryHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "id")
	jobUUID, err := uuid.Parse(jobIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid delivery job UUID")
		return
	}

	job, err := h.queries.GetDeliveryJobByID(r.Context(), pgtype.UUID{Bytes: jobUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Delivery job not found")
		return
	}

	attempts, err := h.queries.ListDeliveryAttemptsByJobID(r.Context(), job.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to load delivery attempts")
		return
	}

	req, _ := h.queries.GetCapturedRequestByID(r.Context(), job.RequestID)

	timelineSteps := []map[string]interface{}{
		{
			"step":        "INGEST",
			"status":      "COMPLETED",
			"timestamp":   req.CreatedAt,
			"description": "Webhook payload received and captured with monotonic ID " + req.RequestID,
		},
		{
			"step":        "SECURITY_INSPECTION",
			"status":      "COMPLETED",
			"timestamp":   req.CreatedAt,
			"description": "HMAC signature and security threat scans evaluated",
		},
	}

	var latestDiagnostic *delivery.DiagnosticResult
	for _, a := range attempts {
		status := "SUCCESS"
		var netErr error
		if a.ErrorMessage.Valid && a.ErrorMessage.String != "" && (!a.ResponseStatusCode.Valid || a.ResponseStatusCode.Int32 == 0) {
			netErr = errors.New(a.ErrorMessage.String)
		}

		diag := delivery.DiagnoseAttempt(int(a.ResponseStatusCode.Int32), netErr, job.TargetUrl, a.ResponseBodySnippet.String)

		if a.ResponseStatusCode.Int32 >= 400 || !a.ResponseStatusCode.Valid || netErr != nil {
			status = "FAILED"
			diagCopy := diag
			latestDiagnostic = &diagCopy
		}

		timelineSteps = append(timelineSteps, map[string]interface{}{
			"step":        "ATTEMPT",
			"attempt":     a.AttemptNumber,
			"status":      status,
			"statusCode":  a.ResponseStatusCode.Int32,
			"latencyMs":   a.LatencyMs,
			"error":       a.ErrorMessage.String,
			"startedAt":   a.StartedAt,
			"finishedAt":  a.FinishedAt,
			"description": "Forwarding attempt " + strconv.Itoa(int(a.AttemptNumber)),
			"diagnostic":  diag,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"job":        job,
		"request":    req,
		"attempts":   attempts,
		"timeline":   timelineSteps,
		"diagnostic": latestDiagnostic,
	})
}

type ReplayRequest struct {
	OverrideIdempotency bool   `json:"overrideIdempotency"`
	Justification       string `json:"justification"`
}

// Replay executes a safe replay of a delivery job.
func (h *DeliveryHandler) Replay(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "id")
	jobUUID, err := uuid.Parse(jobIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid delivery job UUID")
		return
	}

	var input ReplayRequest
	_ = json.NewDecoder(r.Body).Decode(&input)

	// Fetch Job
	job, err := h.queries.GetDeliveryJobByID(r.Context(), pgtype.UUID{Bytes: jobUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Delivery job not found")
		return
	}

	// Audit Logging for Replay
	orgID := middleware.GetOrganizationID(r.Context())
	userID := middleware.GetUserID(r.Context())
	endpoint, _ := h.queries.GetEndpointByIDOnly(r.Context(), job.EndpointID)

	action := "REPLAY_EXECUTED"
	justificationText := input.Justification
	if justificationText == "" {
		justificationText = "Delivery Control Plane üzerinden replay tetiklendi"
	}

	// Idempotency Override Protection
	if input.OverrideIdempotency {
		userRole := middleware.GetRole(r.Context())
		if userRole != "OWNER" {
			writeError(w, http.StatusForbidden, "ROLE_UNAUTHORIZED", "Only organization OWNER can override idempotency for replay")
			return
		}
		if input.Justification == "" {
			writeError(w, http.StatusBadRequest, "JUSTIFICATION_REQUIRED", "A justification is required when overriding idempotency")
			return
		}
		action = "REPLAY_IDEMPOTENCY_OVERRIDDEN"
	}

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"jobId":               jobIDStr,
		"requestId":           uuid.UUID(job.RequestID.Bytes).String(),
		"overrideIdempotency": input.OverrideIdempotency,
		"targetUrl":           job.TargetUrl,
	})

	_, _ = h.queries.CreateAuditLog(r.Context(), database.CreateAuditLogParams{
		OrganizationID: orgID,
		ProjectID:      endpoint.ProjectID,
		UserID:         userID,
		Action:         action,
		ResourceType:   "DELIVERY_JOB",
		ResourceID:     jobIDStr,
		Justification:  pgtype.Text{String: justificationText, Valid: true},
		IpAddress:      pgtype.Text{String: r.RemoteAddr, Valid: true},
		Metadata:       metaJSON,
	})

	// Requeue job to PENDING
	requeued, err := h.queries.RequeueDeliveryJob(r.Context(), job.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to requeue delivery job")
		return
	}

	// Trigger async immediate dispatch
	req, _ := h.queries.GetCapturedRequestByID(r.Context(), requeued.RequestID)
	var headers map[string]string
	_ = json.Unmarshal(req.Headers, &headers)

	h.deliverySvc.ProcessJobAsync(requeued, req.HttpMethod, headers, []byte(req.MaskedBody.String))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Delivery job successfully re-queued for execution",
		"job":     requeued,
	})
}

// GetKPIs returns aggregated metrics for deliveries in a project.
func (h *DeliveryHandler) GetKPIs(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projUUID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project UUID")
		return
	}

	// Calculate KPIs using captured requests and delivery jobs
	endpoints, err := h.queries.ListEndpointsByProject(r.Context(), pgtype.UUID{Bytes: projUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to load project endpoints")
		return
	}

	totalDelivered := 0
	totalDeadLetter := 0
	totalPending := 0
	totalRetryWait := 0

	for _, ep := range endpoints {
		jobs, _ := h.queries.ListDeliveryJobsByEndpoint(r.Context(), database.ListDeliveryJobsByEndpointParams{
			EndpointID: ep.ID,
			Limit:      1000,
			Offset:     0,
		})
		for _, j := range jobs {
			switch delivery.DeliveryState(j.Status) {
			case delivery.DeliveryStateDelivered:
				totalDelivered++
			case delivery.DeliveryStateDeadLetter:
				totalDeadLetter++
			case delivery.DeliveryStatePending:
				totalPending++
			case delivery.DeliveryStateRetryWait:
				totalRetryWait++
			}
		}
	}

	totalAll := totalDelivered + totalDeadLetter + totalPending + totalRetryWait
	successRate := 100.0
	if totalAll > 0 {
		successRate = float64(totalDelivered) / float64(totalAll) * 100.0
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalDeliveries": totalAll,
		"delivered":       totalDelivered,
		"deadLetter":      totalDeadLetter,
		"pending":         totalPending,
		"retryWait":       totalRetryWait,
		"successRate":     successRate,
		"dlqBacklog":      totalDeadLetter,
		"timestamp":       time.Now().Format(time.RFC3339),
	})
}

// ListAuditLogs returns project audit trail.
func (h *DeliveryHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projUUID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project UUID")
		return
	}

	limit := int32(50)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	orgID := middleware.GetOrganizationID(r.Context())
	logs, err := h.queries.ListAuditLogsByProjectOrOrg(r.Context(), database.ListAuditLogsByProjectOrOrgParams{
		ProjectID:      pgtype.UUID{Bytes: projUUID, Valid: true},
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list audit logs")
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list audit logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"auditLogs": logs,
		"count":     len(logs),
	})
}
