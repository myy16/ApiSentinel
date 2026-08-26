package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/id"
	"github.com/apisentinel/apisentinel/internal/policy"
	"github.com/apisentinel/apisentinel/internal/security"
	"github.com/apisentinel/apisentinel/internal/security/duplicate"
	"github.com/apisentinel/apisentinel/internal/security/pii"
	"github.com/apisentinel/apisentinel/internal/security/schema"
	"github.com/apisentinel/apisentinel/internal/valkey"
	"github.com/apisentinel/apisentinel/internal/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type IngestionService struct {
	queries        *database.Queries
	valkeyClient   *valkey.Client
	securityEngine *security.Engine
	alertService   *AlertService
	forwardingSvc  *ForwardingService
	workerPool     *worker.Pool
}

func NewIngestionService(
	queries *database.Queries,
	valkeyClient *valkey.Client,
	alertService *AlertService,
	forwardingSvc *ForwardingService,
	workerPool *worker.Pool,
) *IngestionService {
	if workerPool == nil {
		workerPool = worker.NewPool(10, 2000)
	}
	return &IngestionService{
		queries:        queries,
		valkeyClient:   valkeyClient,
		securityEngine: security.NewEngine(),
		alertService:   alertService,
		forwardingSvc:  forwardingSvc,
		workerPool:     workerPool,
	}
}

type IngestionResult struct {
	StatusCode   int                    `json:"statusCode"`
	ResponseBody map[string]interface{} `json:"responseBody"`
	RequestID    string                 `json:"requestId"`
	Action       string                 `json:"action"`
}

// ProcessWebhook executes the complete ingestion pipeline:
// 1. Resolve & Validate Endpoint
// 2. Generate Time-Sortable UUIDv7 Request ID
// 3. Evaluate Mock Rules (if MOCK mode)
// 4. Security & Injection Inspection
// 5. Idempotency & Duplicate Check
// 6. Schema Contract Validation
// 7. Policy Decision (ALLOW/BLOCK/MASK)
// 8. Mask PII & Secrets before DB storage
// 9. Persist Captured Request
// 10. Worker Pool Async Stream & SSE Publish
// 11. Async Alert Dispatch
// 12. Forwarding / Upstream proxy (if configured)
func (s *IngestionService) ProcessWebhook(
	ctx context.Context,
	slug string,
	httpMethod string,
	headers map[string][]string,
	queryParams map[string][]string,
	rawBody []byte,
	clientIP string,
) (*IngestionResult, error) {
	// 1. Resolve & validate endpoint
	endpoint, err := s.queries.GetEndpointBySlug(ctx, slug)
	if err != nil {
		return nil, errors.New("endpoint bulunamadı")
	}
	if !endpoint.IsActive {
		return nil, errors.New("endpoint pasif durumda")
	}

	// 2. Time-sortable K-Sortable Request ID (UUIDv7)
	requestId := id.NewRequestID()

	headersBytes, _ := json.Marshal(headers)
	queryBytes, _ := json.Marshal(queryParams)

	var parsedJson []byte
	var jsonVal interface{}
	if err := json.Unmarshal(rawBody, &jsonVal); err == nil {
		parsedJson = rawBody
	}

	var pgRawBody pgtype.Text
	if len(rawBody) > 0 {
		pgRawBody = pgtype.Text{String: string(rawBody), Valid: true}
	}

	// 3. Check for Mock Mode & Active Mock Rules
	if endpoint.Mode == "MOCK" {
		return s.handleMockMode(ctx, endpoint, requestId, httpMethod, headersBytes, queryBytes, pgRawBody, parsedJson)
	}

	// 4. Multi-Layer Security Inspection (PII + Secrets + Injection + Obfuscation)
	findings := s.securityEngine.Inspect(string(rawBody))

	// 5. Idempotency & Duplicate Request Detection (Valkey sliding window)
	if s.valkeyClient != nil && len(rawBody) > 0 {
		endpointIdStr := uuid.UUID(endpoint.ID.Bytes).String()
		payloadHash := duplicate.CalculatePayloadHash(rawBody)
		idempKey := duplicate.BuildIdempotencyKey(endpointIdStr, payloadHash)

		if isDup, origReqID, err := s.valkeyClient.CheckAndSetIdempotency(ctx, idempKey, requestId, duplicate.DefaultIdempotencyTTL); err == nil && isDup {
			dupFinding := duplicate.CreateDuplicateFinding(origReqID, payloadHash)
			findings = append(findings, security.Finding{
				Category:       "DUPLICATE",
				Type:           dupFinding.Type,
				Severity:       dupFinding.Severity,
				Message:        dupFinding.Message,
				EvidenceMasked: dupFinding.EvidenceMasked,
				Confidence:     dupFinding.Confidence,
			})
		}
	}

	// 6. JSON Schema Contract Validation
	s.validateSchemaContract(ctx, endpoint.ID, rawBody, &findings)

	// 7. Deterministic Policy Evaluation
	decision := policy.Evaluate(findings)

	var responseStatus int32 = http.StatusOK
	action := string(decision.Action)

	if endpoint.Mode == "BLOCK" || decision.Action == policy.ActionBlock {
		responseStatus = http.StatusForbidden
		action = "BLOCK"
	}

	// 8. Build Masked Body
	maskedBodyStr := s.maskPIIAndSecrets(rawBody, findings)
	var pgMaskedBody pgtype.Text
	if len(maskedBodyStr) > 0 {
		pgMaskedBody = pgtype.Text{String: maskedBodyStr, Valid: true}
	}

	// 9. Persist to PostgreSQL
	captured, err := s.queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
		EndpointID:       endpoint.ID,
		RequestID:        requestId,
		HttpMethod:       httpMethod,
		Headers:          headersBytes,
		QueryParams:      queryBytes,
		RawBody:          pgRawBody,
		MaskedBody:       pgMaskedBody,
		ParsedJson:       parsedJson,
		ResponseStatus:   pgtype.Int4{Int32: responseStatus, Valid: true},
		ProcessingStatus: "RECEIVED",
	})
	if err != nil {
		log.Error().Err(err).Str("requestId", requestId).Msg("Failed to persist captured request")
		return &IngestionResult{
			StatusCode: http.StatusInternalServerError,
			RequestID:  requestId,
			Action:     action,
			ResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":      "PERSISTENCE_FAILED",
					"message":   "Failed to persist captured request",
					"requestId": requestId,
				},
			},
		}, fmt.Errorf("failed to persist captured request: %w", err)
	}

	capturedIdStr := uuid.UUID(captured.ID.Bytes).String()
	projectIdStr := uuid.UUID(endpoint.ProjectID.Bytes).String()
	endpointIdStr := uuid.UUID(endpoint.ID.Bytes).String()

	// 10. Async Worker Pool Stream & SSE Publishing
	s.dispatchAsyncEvents(capturedIdStr, projectIdStr, endpointIdStr, requestId, httpMethod, responseStatus, action, rawBody)

	// 11. Persist Findings & Dispatch Alerts
	s.persistFindingsAndAlerts(ctx, captured, endpoint, findings)

	// 12. Policy Block Response
	if action == "BLOCK" {
		var blockReason string
		for _, f := range findings {
			if f.Category == "INJECTION" || f.Severity == "CRITICAL" {
				blockReason = f.Message
				break
			}
		}
		if blockReason == "" {
			blockReason = "İstek güvenlik politikası gereği engellendi"
		}

		return &IngestionResult{
			StatusCode: http.StatusForbidden,
			RequestID:  requestId,
			Action:     "BLOCK",
			ResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":      "POLICY_BLOCKED",
					"message":   blockReason,
					"requestId": requestId,
				},
			},
		}, nil
	}

	// 13. Forwarding to Upstream Target (if configured)
	if s.forwardingSvc != nil {
		endpointIdStr := uuid.UUID(endpoint.ID.Bytes).String()
		capturedIdStr := uuid.UUID(captured.ID.Bytes).String()
		flatHeaders := make(map[string]string)
		for k, v := range headers {
			if len(v) > 0 {
				flatHeaders[k] = v[0]
			}
		}
		s.forwardingSvc.ForwardCleanWebhook(ctx, endpointIdStr, capturedIdStr, httpMethod, flatHeaders, rawBody)
	}

	return &IngestionResult{
		StatusCode: http.StatusOK,
		RequestID:  requestId,
		Action:     "FORWARD",
		ResponseBody: map[string]interface{}{
			"success":   true,
			"message":   "Webhook accepted",
			"requestId": requestId,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}, nil
}

// --- Pipeline Helper Functions ---

func (s *IngestionService) handleMockMode(
	ctx context.Context,
	endpoint database.Endpoint,
	requestId, httpMethod string,
	headersBytes, queryBytes []byte,
	pgRawBody pgtype.Text,
	parsedJson []byte,
) (*IngestionResult, error) {
	mockRule, mErr := s.queries.GetMatchingMockRule(ctx, endpoint.ID)
	if mErr == nil {
		if mockRule.DelayMs > 0 {
			time.Sleep(time.Duration(mockRule.DelayMs) * time.Millisecond)
		}

		var mockRespBody map[string]interface{}
		_ = json.Unmarshal(mockRule.ResponseBody, &mockRespBody)
		if mockRespBody == nil {
			mockRespBody = map[string]interface{}{"status": "mocked", "rule": mockRule.Name}
		}

		_, _ = s.queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
			EndpointID:       endpoint.ID,
			RequestID:        requestId,
			HttpMethod:       httpMethod,
			Headers:          headersBytes,
			QueryParams:      queryBytes,
			RawBody:          pgRawBody,
			MaskedBody:       pgRawBody,
			ParsedJson:       parsedJson,
			ResponseStatus:   pgtype.Int4{Int32: mockRule.StatusCode, Valid: true},
			ProcessingStatus: "MOCKED",
		})

		return &IngestionResult{
			StatusCode:   int(mockRule.StatusCode),
			RequestID:    requestId,
			Action:       "MOCK",
			ResponseBody: mockRespBody,
		}, nil
	}

	return &IngestionResult{
		StatusCode: http.StatusOK,
		RequestID:  requestId,
		Action:     "MOCK",
		ResponseBody: map[string]interface{}{
			"status":    "mocked",
			"requestId": requestId,
		},
	}, nil
}

func (s *IngestionService) validateSchemaContract(
	ctx context.Context,
	endpointID pgtype.UUID,
	rawBody []byte,
	findings *[]security.Finding,
) {
	schemaRecord, sErr := s.queries.GetEndpointSchema(ctx, endpointID)
	if sErr == nil && len(schemaRecord.JsonSchema) > 0 && len(rawBody) > 0 {
		if validator, vErr := schema.NewValidator(string(schemaRecord.JsonSchema)); vErr == nil {
			if violations, valErr := validator.Validate(rawBody); valErr == nil {
				for _, v := range violations {
					*findings = append(*findings, security.Finding{
						Category:       "CONTRACT",
						Type:           "SCHEMA_VIOLATION",
						Severity:       "HIGH",
						Message:        fmt.Sprintf("JSON Schema ihlali (Alan: %s): %s", v.FieldPath, v.Message),
						EvidenceMasked: v.Keyword,
						Confidence:     1.0,
					})
				}
			}
		}
	}
}

func (s *IngestionService) maskPIIAndSecrets(rawBody []byte, findings []security.Finding) string {
	maskedBodyStr := string(rawBody)
	for _, f := range findings {
		if f.Category == "PII" || f.Category == "SECRET" {
			switch f.Type {
			case "CREDIT_CARD":
				for _, match := range pii.FindCreditCards(maskedBodyStr) {
					maskedBodyStr = strings.Replace(maskedBodyStr, match, pii.MaskCreditCard(match), 1)
				}
			case "TCKN":
				for _, match := range pii.FindTCKNs(maskedBodyStr) {
					maskedBodyStr = strings.Replace(maskedBodyStr, match, pii.MaskTCKN(match), 1)
				}
			case "EMAIL":
				for _, match := range pii.FindEmails(maskedBodyStr) {
					maskedBodyStr = strings.Replace(maskedBodyStr, match, pii.MaskEmail(match), 1)
				}
			case "IBAN":
				for _, match := range pii.FindIBANs(maskedBodyStr) {
					maskedBodyStr = strings.Replace(maskedBodyStr, match, pii.MaskIBAN(match), 1)
				}
			}
		}
	}
	return maskedBodyStr
}

func (s *IngestionService) dispatchAsyncEvents(
	capturedIdStr, projectIdStr, endpointIdStr, requestId, httpMethod string,
	responseStatus int32,
	action string,
	rawBody []byte,
) {
	if s.valkeyClient != nil && s.workerPool != nil {
		_ = s.workerPool.Submit(func(taskCtx context.Context) {
			bgCtx, cancel := context.WithTimeout(taskCtx, 3*time.Second)
			defer cancel()

			s.valkeyClient.PublishStream(bgCtx, "stream:requests", map[string]interface{}{
				"requestId":  capturedIdStr,
				"endpointId": endpointIdStr,
				"projectId":  projectIdStr,
				"rawBody":    string(rawBody),
			})

			eventPayload, _ := json.Marshal(map[string]interface{}{
				"event":          "request.created",
				"id":             capturedIdStr,
				"requestId":      requestId,
				"httpMethod":     httpMethod,
				"responseStatus": responseStatus,
				"action":         action,
				"createdAt":      time.Now().Format(time.RFC3339),
			})
			s.valkeyClient.PublishEvent(bgCtx, "channel:events:"+projectIdStr, string(eventPayload))
		})
	}
}

func (s *IngestionService) persistFindingsAndAlerts(
	ctx context.Context,
	captured database.CapturedRequest,
	endpoint database.Endpoint,
	findings []security.Finding,
) {
	if len(findings) == 0 {
		return
	}

	var dbFindings []database.SecurityFinding
	for _, f := range findings {
		createdFinding, fErr := s.queries.CreateSecurityFinding(ctx, database.CreateSecurityFindingParams{
			RequestID:      captured.ID,
			Category:       f.Category,
			Type:           f.Type,
			Severity:       f.Severity,
			Action:         "LOG",
			FieldPath:      pgtype.Text{Valid: false},
			Message:        f.Message,
			EvidenceMasked: pgtype.Text{String: f.EvidenceMasked, Valid: f.EvidenceMasked != ""},
			Confidence:     pgtype.Float8{Float64: f.Confidence, Valid: true},
		})
		if fErr != nil {
			log.Error().Err(fErr).Str("findingType", f.Type).Msg("Failed to persist security finding")
			continue
		}
		dbFindings = append(dbFindings, createdFinding)
	}

	if s.alertService != nil && len(dbFindings) > 0 {
		projectIdStr := uuid.UUID(endpoint.ProjectID.Bytes).String()
		s.alertService.DispatchForFindings(projectIdStr, endpoint.Name, endpoint.Name, captured.RequestID, dbFindings, "ALERT")
	}
}
