package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/policy"
	"github.com/apisentinel/apisentinel/internal/security"
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

	// 3. Fast Security Inspection & Policy Decision
	findings := s.securityEngine.Inspect(string(rawBody))
	decision := policy.Evaluate(findings)

	var responseStatus int32 = http.StatusOK
	action := string(decision.Action)

	// Handle endpoint mode override or policy block
	if endpoint.Mode == "BLOCK" || decision.Action == policy.ActionBlock {
		responseStatus = http.StatusForbidden
		action = "BLOCK"
	}

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

	// 4. Save to PostgreSQL
	captured, err := s.queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
		EndpointID:       endpoint.ID,
		RequestID:        requestId,
		HttpMethod:       httpMethod,
		Headers:          headersBytes,
		QueryParams:      queryBytes,
		RawBody:          pgRawBody,
		MaskedBody:       pgRawBody,
		ParsedJson:       parsedJson,
		ResponseStatus:   pgtype.Int4{Int32: responseStatus, Valid: true},
		ProcessingStatus: "RECEIVED",
	})
	if err != nil {
		log.Error().Err(err).Str("requestId", requestId).Msg("Failed to persist captured request")
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
