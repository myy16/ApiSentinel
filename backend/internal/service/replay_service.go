package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/ssrf"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type ReplayService struct {
	queries       *database.Queries
	httpClient    *http.Client
	encryptionKey string
}

func NewReplayService(queries *database.Queries, encryptionKey ...string) *ReplayService {
	var key string
	if len(encryptionKey) > 0 {
		key = encryptionKey[0]
	}
	return &ReplayService{
		queries:       queries,
		httpClient:    ssrf.NewSafeHTTPClient(10 * time.Second),
		encryptionKey: key,
	}
}

type ExecuteReplayParams struct {
	SourceRequestId     string            `json:"sourceRequestId"`
	TargetURL           string            `json:"targetUrl"`
	Environment         string            `json:"environment"` // "PRODUCTION", "STAGING", "DEV", "LOCAL", "CUSTOM"
	CustomHeaders       map[string]string `json:"customHeaders,omitempty"`
	Justification       string            `json:"justification,omitempty"`
	OverrideIdempotency bool              `json:"overrideIdempotency"`
	RenewIdempotency    bool              `json:"renewIdempotency"`
	UserID              string            `json:"userId,omitempty"`
	ClientIP            string            `json:"clientIp,omitempty"`
}

type ReplayResultResponse struct {
	JobID                  string            `json:"jobId"`
	Status                 string            `json:"status"`
	ResponseStatus         int               `json:"responseStatus"`
	ResponseBody           string            `json:"responseBody"`
	LatencyMs              int64             `json:"latencyMs"`
	TargetURL              string            `json:"targetUrl"`
	Environment            string            `json:"environment"`
	CustomHeaders          map[string]string `json:"customHeaders,omitempty"`
	OriginalResponseStatus int               `json:"originalResponseStatus,omitempty"`
	Replacements           map[string]string `json:"replacements,omitempty"`
	CreatedAt              string            `json:"createdAt"`
}

func (s *ReplayService) ExecuteReplay(ctx context.Context, params ExecuteReplayParams) (*ReplayResultResponse, error) {
	if params.TargetURL == "" {
		return nil, errors.New("hedef URL (targetUrl) zorunludur")
	}

	if params.Environment == "" {
		params.Environment = "CUSTOM"
	}

	// 1. SSRF Validation
	_, err := ssrf.ValidateURL(params.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("SSRF Guard hedef URL'yi engelledi: %w", err)
	}

	// 2. Fetch original captured request
	reqUUID, err := uuid.Parse(params.SourceRequestId)
	if err != nil {
		return nil, errors.New("geçersiz istek ID formatı")
	}

	var pgReqId pgtype.UUID
	copy(pgReqId.Bytes[:], reqUUID[:])
	pgReqId.Valid = true

	reqRecord, err := s.queries.GetCapturedRequestByID(ctx, pgReqId)
	if err != nil {
		return nil, errors.New("kayıtlı istek bulunamadı")
	}

	var customHeadersJSON []byte
	if len(params.CustomHeaders) > 0 {
		rawJSON, _ := json.Marshal(params.CustomHeaders)
		if s.encryptionKey != "" {
			encVal, encErr := envelope.Encrypt(s.encryptionKey, string(rawJSON))
			if encErr != nil {
				return nil, fmt.Errorf("custom headers could not be encrypted: %w", encErr)
			}
			customHeadersJSON, _ = json.Marshal(map[string]string{"_encrypted": encVal})
		} else {
			customHeadersJSON = rawJSON
		}
	} else {
		customHeadersJSON = []byte("{}")
	}

	// 3. Create initial Replay Job
	job, err := s.queries.CreateReplayJob(ctx, database.CreateReplayJobParams{
		SourceRequestID: pgReqId,
		TargetType:      "DIRECT_HTTP",
		TargetUrl:       pgtype.Text{String: params.TargetURL, Valid: true},
		Environment:     params.Environment,
		CustomHeaders:   customHeadersJSON,
		Status:          "RUNNING",
	})
	if err != nil {
		return nil, fmt.Errorf("replay işi oluşturulamadı: %w", err)
	}

	jobIdStr := uuid.UUID(job.ID.Bytes).String()

	// 4. Prepare Outbound Payload & Headers (Prioritize MaskedBody over RawBody for PII safety)
	var rawPayloadBytes []byte
	if reqRecord.MaskedBody.Valid && len(reqRecord.MaskedBody.String) > 0 {
		rawPayloadBytes = []byte(reqRecord.MaskedBody.String)
	} else if len(reqRecord.ParsedJson) > 0 {
		rawPayloadBytes = reqRecord.ParsedJson
	} else if reqRecord.RawBody.Valid && len(reqRecord.RawBody.String) > 0 {
		rawPayloadBytes = []byte(reqRecord.RawBody.String)
	}

	var headers map[string]interface{}
	_ = json.Unmarshal(reqRecord.Headers, &headers)

	var replacements map[string]string
	if params.RenewIdempotency {
		mutation := MutateIdempotencyKeys(headers, rawPayloadBytes)
		rawPayloadBytes = mutation.PayloadBytes
		replacements = mutation.Replacements
		headers = make(map[string]interface{})
		for hk, hv := range mutation.Headers {
			headers[hk] = hv
		}
	}

	outboundReq, err := http.NewRequestWithContext(ctx, reqRecord.HttpMethod, params.TargetURL, bytes.NewBuffer(rawPayloadBytes))
	if err != nil {
		return nil, err
	}

	// Restore / set headers
	for k, v := range headers {
		switch val := v.(type) {
		case string:
			outboundReq.Header.Set(k, val)
		case []interface{}:
			for i, item := range val {
				if strItem, ok := item.(string); ok {
					if i == 0 {
						outboundReq.Header.Set(k, strItem)
					} else {
						outboundReq.Header.Add(k, strItem)
					}
				}
			}
		}
	}

	// Inject custom headers & environment headers
	for hk, hv := range params.CustomHeaders {
		outboundReq.Header.Set(hk, hv)
	}
	outboundReq.Header.Set("X-ApiSentinel-Replayed", "true")
	outboundReq.Header.Set("X-ApiSentinel-Environment", params.Environment)

	// 5. Execute HTTP Replay
	startTime := time.Now()
	resp, reqErr := s.httpClient.Do(outboundReq)
	latencyMs := time.Since(startTime).Milliseconds()

	var respStatus int = 0
	var respBodyStr string = ""
	status := "COMPLETED"

	if reqErr != nil {
		status = "FAILED"
		respBodyStr = reqErr.Error()
		log.Warn().Err(reqErr).Str("jobId", jobIdStr).Msg("Replay request failed")
	} else {
		respStatus = resp.StatusCode
		defer resp.Body.Close()
		// Limit response body reading to 1 MB to prevent unbounded memory consumption
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		respBodyStr = string(bodyBytes)
	}

	// Truncate DB stored response body to 64 KB
	dbRespBody := respBodyStr
	if len(dbRespBody) > 65536 {
		dbRespBody = dbRespBody[:65536] + "... [TRUNCATED]"
	}

	// 6. Update Replay Job with Results
	now := time.Now()
	updatedJob, err := s.queries.UpdateReplayJobResult(ctx, database.UpdateReplayJobResultParams{
		ID:             job.ID,
		Status:         status,
		ResponseStatus: pgtype.Int4{Int32: int32(respStatus), Valid: true},
		ResponseBody:   pgtype.Text{String: dbRespBody, Valid: true},
		LatencyMs:      int32(latencyMs),
		CompletedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to update replay job record")
	}

	// 7. Record to Audit Trail
	orgID, err := s.queries.GetProjectOrganizationID(ctx, reqRecord.ProjectID)
	if err != nil {
		log.Warn().Err(err).Msg("Could not fetch project organization ID for audit log")
	}

	justificationText := params.Justification
	if justificationText == "" {
		justificationText = fmt.Sprintf("Replay Lab üzerinden %s hedefine (%s) tekrar iletildi", params.Environment, params.TargetURL)
	}

	var originalStatus int32 = 0
	if reqRecord.ResponseStatus.Valid {
		originalStatus = reqRecord.ResponseStatus.Int32
	}

	action := "REPLAY_LAB_EXECUTED"
	if params.OverrideIdempotency {
		action = "REPLAY_IDEMPOTENCY_OVERRIDDEN"
	}

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"targetUrl":              params.TargetURL,
		"environment":            params.Environment,
		"responseStatus":         respStatus,
		"originalResponseStatus": originalStatus,
		"latencyMs":              latencyMs,
		"status":                 status,
		"replayedFrom":           "REPLAY_LAB",
		"overrideIdempotency":    params.OverrideIdempotency,
		"renewIdempotency":       params.RenewIdempotency,
	})

	var pgUserId pgtype.UUID
	if params.UserID != "" {
		if uUUID, err := uuid.Parse(params.UserID); err == nil {
			pgUserId = pgtype.UUID{Bytes: uUUID, Valid: true}
		}
	}

	_, auditErr := s.queries.CreateAuditLog(ctx, database.CreateAuditLogParams{
		OrganizationID: orgID,
		ProjectID:      reqRecord.ProjectID,
		UserID:         pgUserId,
		Action:         action,
		ResourceType:   "CAPTURED_REQUEST",
		ResourceID:     reqRecord.RequestID,
		Justification:  pgtype.Text{String: justificationText, Valid: true},
		IpAddress:      pgtype.Text{String: params.ClientIP, Valid: params.ClientIP != ""},
		Metadata:       metaJSON,
	})
	if auditErr != nil {
		log.Error().Err(auditErr).Msg("Failed to write audit log in ReplayService")
	}

	return &ReplayResultResponse{
		JobID:                  jobIdStr,
		Status:                 updatedJob.Status,
		ResponseStatus:         respStatus,
		ResponseBody:           respBodyStr,
		LatencyMs:              latencyMs,
		TargetURL:              params.TargetURL,
		Environment:            params.Environment,
		CustomHeaders:          params.CustomHeaders,
		OriginalResponseStatus: int(originalStatus),
		Replacements:           replacements,
		CreatedAt:              updatedJob.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *ReplayService) ListReplayJobs(ctx context.Context, projectId string, limit, offset int32) ([]map[string]interface{}, error) {
	projUUID, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], projUUID[:])
	pgProjId.Valid = true

	jobs, err := s.queries.ListReplayJobsByProject(ctx, database.ListReplayJobsByProjectParams{
		ProjectID: pgProjId,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	var res []map[string]interface{}
	for _, j := range jobs {
		var status int32 = 0
		if j.ResponseStatus.Valid {
			status = j.ResponseStatus.Int32
		}
		var origStatus int32 = 0
		if j.OriginalResponseStatus.Valid {
			origStatus = j.OriginalResponseStatus.Int32
		}
		var target string = ""
		if j.TargetUrl.Valid {
			target = j.TargetUrl.String
		}
		var respBody string = ""
		if j.ResponseBody.Valid {
			respBody = j.ResponseBody.String
		}

		customH := s.resolveAndMaskCustomHeaders(j.CustomHeaders)

		res = append(res, map[string]interface{}{
			"id":                     uuid.UUID(j.ID.Bytes).String(),
			"sourceRequestId":        uuid.UUID(j.SourceRequestID.Bytes).String(),
			"requestId":              j.RequestID,
			"httpMethod":             j.HttpMethod,
			"endpointName":           j.EndpointName,
			"targetUrl":              target,
			"environment":            j.Environment,
			"customHeaders":          customH,
			"status":                 j.Status,
			"responseStatus":         status,
			"originalResponseStatus": origStatus,
			"latencyMs":              j.LatencyMs,
			"responseBody":           respBody,
			"createdAt":              j.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}

func (s *ReplayService) GetReplayJob(ctx context.Context, replayId string) (map[string]interface{}, error) {
	rUUID, err := uuid.Parse(replayId)
	if err != nil {
		return nil, errors.New("geçersiz replay ID formatı")
	}

	job, err := s.queries.GetReplayJobByID(ctx, pgtype.UUID{Bytes: rUUID, Valid: true})
	if err != nil {
		return nil, errors.New("replay kaydı bulunamadı")
	}

	customH := s.resolveAndMaskCustomHeaders(job.CustomHeaders)

	return map[string]interface{}{
		"id":                     uuid.UUID(job.ID.Bytes).String(),
		"sourceRequestId":        uuid.UUID(job.SourceRequestID.Bytes).String(),
		"requestId":              job.RequestID,
		"httpMethod":             job.HttpMethod,
		"endpointName":           job.EndpointName,
		"endpointSlug":           job.EndpointSlug,
		"targetUrl":              job.TargetUrl.String,
		"environment":            job.Environment,
		"customHeaders":          customH,
		"status":                 job.Status,
		"responseStatus":         job.ResponseStatus.Int32,
		"originalResponseStatus": job.OriginalResponseStatus.Int32,
		"latencyMs":              job.LatencyMs,
		"responseBody":           job.ResponseBody.String,
		"originalPayload":        string(job.OriginalPayload),
		"createdAt":              job.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

// resolveAndMaskCustomHeaders decrypts custom headers if encrypted, then masks sensitive tokens
func (s *ReplayService) resolveAndMaskCustomHeaders(raw []byte) map[string]string {
	if len(raw) == 0 {
		return make(map[string]string)
	}

	var headers map[string]string
	var env map[string]string

	if err := json.Unmarshal(raw, &env); err == nil {
		if encVal, ok := env["_encrypted"]; ok && encVal != "" {
			if s.encryptionKey != "" {
				if dec, dErr := envelope.Decrypt(s.encryptionKey, encVal); dErr == nil && dec != "" {
					_ = json.Unmarshal([]byte(dec), &headers)
				}
			}
		} else {
			headers = env
		}
	} else if s.encryptionKey != "" {
		if dec, err := envelope.Decrypt(s.encryptionKey, string(raw)); err == nil && dec != "" {
			_ = json.Unmarshal([]byte(dec), &headers)
		}
	}

	if headers == nil {
		_ = json.Unmarshal(raw, &headers)
	}

	masked := make(map[string]string)
	for k, v := range headers {
		masked[k] = envelope.MaskHeaderValue(k, v)
	}
	return masked
}
