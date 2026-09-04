package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/apisentinel/apisentinel/internal/ai"
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
	explainer   *ai.Explainer
}

func NewDeliveryHandler(queries *database.Queries, deliverySvc *service.DeliveryService, explainer *ai.Explainer) *DeliveryHandler {
	return &DeliveryHandler{
		queries:     queries,
		deliverySvc: deliverySvc,
		explainer:   explainer,
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
			if limit > 200 {
				limit = 200
			}
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

	// Trigger queue poller for lease-locked atomic execution
	h.deliverySvc.TriggerQueue()

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

	// Calculate KPIs using CountDeliveryJobsByProjectAndStatus query directly in DB
	counts, err := h.queries.CountDeliveryJobsByProjectAndStatus(r.Context(), pgtype.UUID{Bytes: projUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to compute delivery KPIs")
		return
	}

	totalDelivered := int64(0)
	totalDeadLetter := int64(0)
	totalPending := int64(0)
	totalRetryWait := int64(0)

	for _, c := range counts {
		switch delivery.DeliveryState(c.Status) {
		case delivery.DeliveryStateDelivered:
			totalDelivered += c.Count
		case delivery.DeliveryStateDeadLetter:
			totalDeadLetter += c.Count
		case delivery.DeliveryStatePending:
			totalPending += c.Count
		case delivery.DeliveryStateRetryWait:
			totalRetryWait += c.Count
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

// AIExplain generates root-cause diagnosis and actionable fix for a failed delivery job.
func (h *DeliveryHandler) AIExplain(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "id")
	jobUUID, err := uuid.Parse(jobIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "Geçersiz delivery job ID")
		return
	}

	job, err := h.queries.GetDeliveryJobByID(r.Context(), pgtype.UUID{Bytes: jobUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Teslimat kaydı bulunamadı")
		return
	}

	endpoint, err := h.queries.GetEndpointByIDOnly(r.Context(), job.EndpointID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint bulunamadı")
		return
	}

	// 1. Get captured request for payload and method
	capReq, _ := h.queries.GetCapturedRequestByID(r.Context(), job.RequestID)

	// 2. Get latest attempt
	attempts, _ := h.queries.ListDeliveryAttemptsByJobID(r.Context(), job.ID)
	var latestAttempt database.DeliveryAttempt
	if len(attempts) > 0 {
		latestAttempt = attempts[len(attempts)-1]
	}

	// 3. Organization AI Settings & Opt-in Guard (Fail-Closed)
	privacyLevel := "FULL_LOCAL"
	var customRedactKeys []string
	orgID := middleware.GetOrganizationID(r.Context())
	if !orgID.Valid {
		writeError(w, http.StatusUnauthorized, "ORGANIZATION_REQUIRED", "Organizasyon kimliği bulunamadı")
		return
	}
	orgSettings, orgErr := h.queries.GetOrganizationAISettings(r.Context(), orgID)
	if orgErr != nil {
		writeError(w, http.StatusInternalServerError, "AI_POLICY_ERROR", "Organizasyon AI güvenlik ayarları okunamadı")
		return
	}
	if !orgSettings.AiEnabled {
		writeError(w, http.StatusForbidden, "AI_DISABLED", "Bu organizasyon için AI analizi devre dışı bırakılmıştır")
		return
	}
	if orgSettings.AiDataSharingLevel != "" {
		privacyLevel = orgSettings.AiDataSharingLevel
	}
	if len(orgSettings.AiCustomRedactionPatterns) > 0 {
		_ = json.Unmarshal(orgSettings.AiCustomRedactionPatterns, &customRedactKeys)
	}

	rawBody := ""
	if capReq.RawBody.Valid {
		rawBody = capReq.RawBody.String
	} else if capReq.MaskedBody.Valid {
		rawBody = capReq.MaskedBody.String
	}

	responseBody := ""
	if latestAttempt.ResponseBodySnippet.Valid {
		responseBody = latestAttempt.ResponseBodySnippet.String
	}

	errorMsg := ""
	if latestAttempt.ErrorMessage.Valid {
		errorMsg = latestAttempt.ErrorMessage.String
	} else if job.LastError.Valid {
		errorMsg = job.LastError.String
	}

	httpMethod := "POST"
	if capReq.HttpMethod != "" {
		httpMethod = capReq.HttpMethod
	}

	status := 0
	if latestAttempt.ResponseStatusCode.Valid {
		status = int(latestAttempt.ResponseStatusCode.Int32)
	}

	if h.explainer == nil {
		h.explainer = ai.NewExplainer("")
	}

	analysis, err := h.explainer.ExplainDeliveryIncident(r.Context(), ai.DeliveryIncidentInput{
		EndpointSlug:     endpoint.Slug,
		HTTPMethod:       httpMethod,
		ResponseStatus:   status,
		ErrorMessage:     errorMsg,
		RequestBody:      rawBody,
		ResponseBody:     responseBody,
		LatencyMs:        int64(latestAttempt.LatencyMs),
		AttemptCount:     int(job.Attempts),
		PrivacyLevel:     privacyLevel,
		CustomRedactKeys: customRedactKeys,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, analysis)
}
