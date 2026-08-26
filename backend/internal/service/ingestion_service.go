package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/policy"
	"github.com/apisentinel/apisentinel/internal/security"
	"github.com/apisentinel/apisentinel/internal/security/pii"
	"github.com/apisentinel/apisentinel/internal/security/schema"
	"github.com/apisentinel/apisentinel/internal/valkey"
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
}

func NewIngestionService(
	queries *database.Queries,
	valkeyClient *valkey.Client,
	alertService *AlertService,
	forwardingSvc *ForwardingService,
) *IngestionService {
	return &IngestionService{
		queries:        queries,
		valkeyClient:   valkeyClient,
		securityEngine: security.NewEngine(),
		alertService:   alertService,
		forwardingSvc:  forwardingSvc,
	}
}

type IngestionResult struct {
	StatusCode   int                    `json:"statusCode"`
	ResponseBody map[string]interface{} `json:"responseBody"`
	RequestID    string                 `json:"requestId"`
	Action       string                 `json:"action"`
}

func (s *IngestionService) ProcessWebhook(
	ctx context.Context,
	slug string,
	httpMethod string,
	headers map[string][]string,
	queryParams map[string][]string,
	rawBody []byte,
	clientIP string,
) (*IngestionResult, error) {
	// 1. Resolve endpoint
	endpoint, err := s.queries.GetEndpointBySlug(ctx, slug)
	if err != nil {
		return nil, errors.New("endpoint bulunamadı")
	}

	if !endpoint.IsActive {
		return nil, errors.New("endpoint pasif durumda")
	}

	// 2. Generate unique RequestID
	b := make([]byte, 8)
	rand.Read(b)
	requestId := "req_" + hex.EncodeToString(b)

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

	// 2.5. Check for Mock Mode & Active Mock Rules
	if endpoint.Mode == "MOCK" {
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

			// Persist captured request as MOCKED
			s.queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
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
	}

	// 3. Fast Security Inspection (PII + Secrets)
	findings := s.securityEngine.Inspect(string(rawBody))

	// 3.5. JSON Schema Contract Validation (if schema is attached to endpoint)
	schemaRecord, sErr := s.queries.GetEndpointSchema(ctx, endpoint.ID)
	if sErr == nil && len(schemaRecord.JsonSchema) > 0 && len(rawBody) > 0 {
		if validator, vErr := schema.NewValidator(string(schemaRecord.JsonSchema)); vErr == nil {
			if violations, valErr := validator.Validate(rawBody); valErr == nil {
				for _, v := range violations {
					findings = append(findings, security.Finding{
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

	decision := policy.Evaluate(findings)

	var responseStatus int32 = http.StatusOK
	action := string(decision.Action)

	// Handle endpoint mode override or policy block
	if endpoint.Mode == "BLOCK" || decision.Action == policy.ActionBlock {
		responseStatus = http.StatusForbidden
		action = "BLOCK"
	}

	// 4. Build masked body — replace detected PII/secret values with masks before storage
	maskedBodyStr := string(rawBody)
	for _, f := range findings {
		if f.Category == "PII" || f.Category == "SECRET" {
			// The evidence_masked field contains the masked version; find and replace original matches
			// We use the scanner's mask functions for known types
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
			default:
				if f.Category == "SECRET" && f.EvidenceMasked != "" {
					// For secrets, evidence_masked has the masked form — try to find the original
					// We can't reverse the mask, but we can use the secret scanner's approach
					// to find matches and replace them
				}
			}
		}
	}

	var pgMaskedBody pgtype.Text
	if len(maskedBodyStr) > 0 {
		pgMaskedBody = pgtype.Text{String: maskedBodyStr, Valid: true}
	}

	// 5. Save to PostgreSQL
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
		// Return early — downstream operations depend on captured.ID being valid
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

	// 5. Asynchronously push to Valkey Stream for worker pipeline
	if s.valkeyClient != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			s.valkeyClient.PublishStream(bgCtx, "stream:requests", map[string]interface{}{
				"requestId":  capturedIdStr,
				"endpointId": endpointIdStr,
				"projectId":  projectIdStr,
				"rawBody":    string(rawBody),
			})

			// 6. Publish to SSE Pub/Sub for realtime dashboard pulse
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
		}()
	}

	// 7. Persist Security Findings & Dispatch Alerts (If findings exist)
	if len(findings) > 0 {
		var dbFindings []database.SecurityFinding
		for _, f := range findings {
			// Persist to security_findings table
			createdFinding, fErr := s.queries.CreateSecurityFinding(ctx, database.CreateSecurityFindingParams{
				RequestID:      captured.ID,
				Category:       f.Category,
				Type:           f.Type,
				Severity:       f.Severity,
				Action:         action,
				Message:        f.Message,
				EvidenceMasked: pgtype.Text{String: f.EvidenceMasked, Valid: true},
				Confidence:     pgtype.Float8{Float64: f.Confidence, Valid: true},
			})
			if fErr != nil {
				log.Error().Err(fErr).Msg("Failed to persist security finding")
				// Skip adding to alert list — zero-value ID would cause issues downstream
				continue
			}

			dbFindings = append(dbFindings, database.SecurityFinding{
				ID:             createdFinding.ID,
				RequestID:      captured.ID,
				Category:       f.Category,
				Type:           f.Type,
				Severity:       f.Severity,
				Action:         action,
				Message:        f.Message,
				EvidenceMasked: pgtype.Text{String: f.EvidenceMasked, Valid: true},
				Confidence:     pgtype.Float8{Float64: f.Confidence, Valid: true},
			})
		}

		if s.alertService != nil {
			s.alertService.DispatchForFindings(projectIdStr, "ApiSentinel Project", endpoint.Name, requestId, dbFindings, action)
		}
	}

	// 8. Upstream Forwarding (If clean webhook)
	if s.forwardingSvc != nil && action != "BLOCK" {
		flatHeaders := make(map[string]string)
		for k, v := range headers {
			if len(v) > 0 {
				flatHeaders[k] = v[0]
			}
		}
		s.forwardingSvc.ForwardCleanWebhook(ctx, endpointIdStr, capturedIdStr, httpMethod, flatHeaders, rawBody)
	}

	if responseStatus == http.StatusForbidden {
		return &IngestionResult{
			StatusCode: http.StatusForbidden,
			RequestID:  requestId,
			Action:     action,
			ResponseBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":      "POLICY_BLOCKED",
					"message":   decision.Reason,
					"requestId": requestId,
				},
			},
		}, nil
	}

	return &IngestionResult{
		StatusCode: http.StatusOK,
		RequestID:  requestId,
		Action:     action,
		ResponseBody: map[string]interface{}{
			"success":   true,
			"message":   "Webhook accepted",
			"requestId": requestId,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}, nil
}
