package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/id"
	"github.com/apisentinel/apisentinel/internal/policy"
	"github.com/apisentinel/apisentinel/internal/security"
	"github.com/apisentinel/apisentinel/internal/security/duplicate"
	"github.com/apisentinel/apisentinel/internal/security/hmac"
	"github.com/apisentinel/apisentinel/internal/security/ratelimit"
	"github.com/apisentinel/apisentinel/internal/security/redaction"
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
	rateLimiter    *ratelimit.Limiter
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
	var limiter *ratelimit.Limiter
	if valkeyClient != nil {
		limiter = ratelimit.NewLimiter(valkeyClient)
	}
	return &IngestionService{
		queries:        queries,
		valkeyClient:   valkeyClient,
		rateLimiter:    limiter,
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
// 3. Rate Limit Protection (Valkey Token Bucket)
// 4. Webhook HMAC Signature Verification
// 5. Evaluate Mock Rules (if MOCK mode)
// 6. Security & Injection Inspection
// 7. Idempotency & Duplicate Check
// 8. Schema Contract Validation
// 9. Policy Decision (ALLOW/BLOCK/MASK)
// 10. Mask PII & Secrets before DB storage
// 11. Persist Captured Request
// 12. Worker Pool Async Stream & SSE Publish
// 13. Async Alert Dispatch
// 14. Forwarding / Upstream proxy (if configured)
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

	// 3. Rate Limit Protection (Valkey Token Bucket)
	if s.rateLimiter != nil {
		endpointIdStr := uuid.UUID(endpoint.ID.Bytes).String()
		rateKey := fmt.Sprintf("%s:%s", endpointIdStr, clientIP)
		res, _ := s.rateLimiter.Allow(ctx, rateKey, 120, time.Minute)
		if !res.Allowed {
			return &IngestionResult{
				StatusCode: http.StatusTooManyRequests,
				RequestID:  requestId,
				Action:     "BLOCK",
				ResponseBody: map[string]interface{}{
					"error": map[string]interface{}{
						"code":      "RATE_LIMIT_EXCEEDED",
						"message":   "Too many requests. Rate limit exceeded (120 req/min).",
						"requestId": requestId,
					},
				},
			}, nil
		}
	}

	// 4. Webhook HMAC Signature Verification (if provider signatures are provided)
	if hmacErr := s.verifyWebhookHMAC(endpoint, rawBody, headers); hmacErr != nil {
		return &IngestionResult{
			StatusCode: http.StatusUnauthorized,
			RequestID:  requestId,
			Action:     "BLOCK",
			ResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":      "INVALID_WEBHOOK_SIGNATURE",
					"message":   hmacErr.Error(),
					"requestId": requestId,
				},
			},
		}, nil
	}

	headersBytes, _ := json.Marshal(redaction.Headers(headers))
	queryBytes, _ := json.Marshal(redaction.QueryParams(queryParams))
	maskedBodyStr, parsedJson := redaction.Payload(rawBody)
	var pgMaskedBody pgtype.Text
	if maskedBodyStr != "" {
		pgMaskedBody = pgtype.Text{String: maskedBodyStr, Valid: true}
	}

	// 5. Check for Mock Mode & Active Mock Rules
	if endpoint.Mode == "MOCK" {
		return s.handleMockMode(ctx, endpoint, requestId, httpMethod, headersBytes, queryBytes, pgMaskedBody, parsedJson)
	}

	// 6. Multi-Layer Security Inspection (PII + Secrets + Injection + Obfuscation)
	findings := s.securityEngine.Inspect(string(rawBody))

	// 7. Idempotency & Duplicate Request Detection (Valkey sliding window)
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

	// 8. JSON Schema Contract Validation
	s.validateSchemaContract(ctx, endpoint.ID, rawBody, &findings)

	// 9. Deterministic Policy Evaluation
	decision := policy.Evaluate(findings)

	var responseStatus int32 = http.StatusOK
	action := string(decision.Action)

	if endpoint.Mode == "BLOCK" || decision.Action == policy.ActionBlock {
		responseStatus = http.StatusForbidden
		action = "BLOCK"
	}

	// 10. Persist only redacted payload data to PostgreSQL.
	captured, err := s.queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
		EndpointID:       endpoint.ID,
		RequestID:        requestId,
		HttpMethod:       httpMethod,
		Headers:          headersBytes,
		QueryParams:      queryBytes,
		RawBody:          pgtype.Text{},
		MaskedBody:       pgMaskedBody,
		ParsedJson:       parsedJson,
		ClientIp:         parseClientIP(clientIP),
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

	// 11. Async Worker Pool Stream & SSE Publishing
	s.dispatchAsyncEvents(capturedIdStr, projectIdStr, endpointIdStr, requestId, httpMethod, responseStatus, action)

	// 12. Persist Findings & Dispatch Alerts
	s.persistFindingsAndAlerts(ctx, captured, endpoint, findings)

	// 13. Policy Block Response
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

	// 14. Forwarding to Upstream Target (if configured)
	if s.forwardingSvc != nil {
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

func parseClientIP(raw string) *netip.Addr {
	ip, err := netip.ParseAddr(raw)
	if err != nil {
		return nil
	}
	return &ip
}

// --- Pipeline Helper Functions ---

func (s *IngestionService) verifyWebhookHMAC(
	endpoint database.Endpoint,
	rawBody []byte,
	headers map[string][]string,
) error {
	// 1. Check for Stripe signature
	if hasHeader(headers, "stripe-signature") {
		secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if secret != "" {
			return hmac.Verify(hmac.ProviderStripe, secret, rawBody, headers, 5*time.Minute)
		}
	}

	// 2. Check for GitHub signature
	if hasHeader(headers, "x-hub-signature-256") {
		secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
		if secret != "" {
			return hmac.Verify(hmac.ProviderGitHub, secret, rawBody, headers, 0)
		}
	}

	// 3. Check for Shopify signature
	if hasHeader(headers, "x-shopify-hmac-sha256") {
		secret := os.Getenv("SHOPIFY_WEBHOOK_SECRET")
		if secret != "" {
			return hmac.Verify(hmac.ProviderShopify, secret, rawBody, headers, 0)
		}
	}

	// 4. Check for Generic X-Signature
	if hasHeader(headers, "x-signature", "x-webhook-signature") {
		secret := os.Getenv("GENERIC_WEBHOOK_SECRET")
		if secret != "" {
			return hmac.Verify(hmac.ProviderGeneric, secret, rawBody, headers, 0)
		}
	}

	return nil
}

func hasHeader(headers map[string][]string, keys ...string) bool {
	for _, k := range keys {
		for hk, vals := range headers {
			if strings.EqualFold(hk, k) && len(vals) > 0 {
				return true
			}
		}
	}
	return false
}

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

func (s *IngestionService) dispatchAsyncEvents(
	capturedIdStr, projectIdStr, endpointIdStr, requestId, httpMethod string,
	responseStatus int32,
	action string,
) {
	if s.valkeyClient != nil && s.workerPool != nil {
		_ = s.workerPool.Submit(func(taskCtx context.Context) {
			bgCtx, cancel := context.WithTimeout(taskCtx, 3*time.Second)
			defer cancel()

			s.valkeyClient.PublishStream(bgCtx, "stream:requests", map[string]interface{}{
				"requestId":  capturedIdStr,
				"endpointId": endpointIdStr,
				"projectId":  projectIdStr,
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
