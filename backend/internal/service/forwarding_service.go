package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/forwarding"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/apisentinel/apisentinel/internal/security/redaction"
	"github.com/apisentinel/apisentinel/internal/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type ForwardingService struct {
	queries       *database.Queries
	forwarder     *forwarding.Forwarder
	workerPool    *worker.Pool
	workerID      string
	encryptionKey string
}

func NewForwardingService(queries *database.Queries, workerPool *worker.Pool, encryptionKey ...string) *ForwardingService {
	workerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	encKey := ""
	if len(encryptionKey) > 0 {
		encKey = encryptionKey[0]
	}

	svc := &ForwardingService{
		queries:       queries,
		forwarder:     forwarding.NewForwarder(),
		workerPool:    workerPool,
		workerID:      workerID,
		encryptionKey: encKey,
	}

	// Recover any stale jobs left in PROCESSING on startup (#2.2, #9)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := queries.RecoverStaleOutboxJobs(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to recover stale outbox jobs on startup")
		} else {
			log.Info().Msg("Outbox recovery checked and stale jobs unlocked")
		}
	}()

	return svc
}

type SaveForwardingConfigInput struct {
	EndpointID    string            `json:"endpointId"`
	TargetURL     string            `json:"targetUrl"`
	MaxRetries    int               `json:"maxRetries"`
	TimeoutMs     int               `json:"timeoutMs"`
	CustomHeaders map[string]string `json:"customHeaders"`
	IsEnabled     bool              `json:"isEnabled"`
	PayloadMode   string            `json:"payloadMode"` // "REDACTED" (default) or "RAW"
}

func (s *ForwardingService) SaveConfig(ctx context.Context, input SaveForwardingConfigInput) (*database.ForwardingConfig, error) {
	epUUID, err := uuid.Parse(input.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}

	headersJSON, _ := json.Marshal(input.CustomHeaders)
	if s.encryptionKey != "" && len(input.CustomHeaders) > 0 {
		encrypted, err := envelope.Encrypt(s.encryptionKey, string(headersJSON))
		if err != nil {
			return nil, fmt.Errorf("custom headers could not be encrypted: %w", err)
		}
		if encrypted != "" {
			envelopePayload, _ := json.Marshal(map[string]string{"_encrypted": encrypted})
			headersJSON = envelopePayload
		}
	}

	maxRetries := input.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	timeoutMs := input.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	payloadMode := strings.ToUpper(strings.TrimSpace(input.PayloadMode))
	if payloadMode != "RAW" {
		payloadMode = "REDACTED"
	}

	cfg, err := s.queries.UpsertForwardingConfig(ctx, database.UpsertForwardingConfigParams{
		EndpointID:    pgtype.UUID{Bytes: epUUID, Valid: true},
		TargetUrl:     input.TargetURL,
		MaxRetries:    int32(maxRetries),
		TimeoutMs:     int32(timeoutMs),
		CustomHeaders: headersJSON,
		IsEnabled:     input.IsEnabled,
		PayloadMode:   payloadMode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save forwarding config: %w", err)
	}

	return &cfg, nil
}

func (s *ForwardingService) GetConfig(ctx context.Context, endpointID string) (*database.ForwardingConfig, error) {
	epUUID, err := uuid.Parse(endpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}

	cfg, err := s.queries.GetForwardingConfigByEndpoint(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
	if err != nil {
		return nil, err
	}

	// Decrypt and mask sensitive custom headers for API response (#3.1, #3.2)
	headersMap := s.resolveCustomHeaders(cfg)
	maskedHeaders := make(map[string]string)
	for k, v := range headersMap {
		maskedHeaders[k] = envelope.MaskHeaderValue(k, v)
	}
	cfg.CustomHeaders, _ = json.Marshal(maskedHeaders)

	return &cfg, nil
}

// ForwardCleanWebhook persists job to durable outbox and submits execution to worker pool
func (s *ForwardingService) ForwardCleanWebhook(ctx context.Context, endpointID string, reqID string, method string, headers map[string]string, body []byte) {
	epUUID, err := uuid.Parse(endpointID)
	if err != nil {
		return
	}
	reqUUID, err := uuid.Parse(reqID)
	if err != nil {
		return
	}

	// 1. Resolve forwarding target and config
	cfg, err := s.queries.GetForwardingConfigByEndpoint(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
	targetURL := ""
	maxRetries := int32(3)
	payloadMode := "REDACTED"

	if err == nil && cfg.IsEnabled && cfg.TargetUrl != "" {
		targetURL = cfg.TargetUrl
		maxRetries = cfg.MaxRetries
		if cfg.PayloadMode != "" {
			payloadMode = cfg.PayloadMode
		}
	} else {
		// Fallback to endpoint's upstream_url
		endpoint, epErr := s.queries.GetEndpointByIDOnly(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
		if epErr == nil && endpoint.UpstreamUrl.Valid && endpoint.UpstreamUrl.String != "" {
			targetURL = endpoint.UpstreamUrl.String
		}
	}

	if targetURL == "" {
		return
	}

	// 2. Prepare payload based on payloadMode (#8: REDACTED is strict default)
	var finalPayload string
	if payloadMode == "RAW" {
		log.Warn().
			Str("endpointId", endpointID).
			Str("requestId", reqID).
			Msg("AUDIT: Webhook forwarded in RAW unredacted mode")
		finalPayload = string(body)
	} else {
		maskedPayload, _ := redaction.Payload(body)
		finalPayload = maskedPayload
	}

	// 3. Create durable Outbox job in database (#2.1, #9: PENDING state)
	outboxJob, err := s.queries.CreateOutboxJob(ctx, database.CreateOutboxJobParams{
		EndpointID:  pgtype.UUID{Bytes: epUUID, Valid: true},
		RequestID:   pgtype.UUID{Bytes: reqUUID, Valid: true},
		TargetUrl:   targetURL,
		Payload:     pgtype.Text{String: finalPayload, Valid: finalPayload != ""},
		PayloadMode: payloadMode,
		MaxRetries:  maxRetries,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to persist outbox job in database")
		return
	}

	// 4. Dispatch job to worker pool
	task := func(taskCtx context.Context) {
		bgCtx := taskCtx
		if bgCtx == nil {
			bgCtx = context.Background()
		}

		s.executeOutboxJob(bgCtx, outboxJob, method, headers, []byte(finalPayload))
	}

	if s.workerPool != nil {
		if err := s.workerPool.Submit(task); err != nil {
			log.Warn().Err(err).Msg("Worker pool full; outbox job remains PENDING for next queue batch")
		}
	}
}

func (s *ForwardingService) executeOutboxJob(ctx context.Context, job database.ForwardingDlq, method string, headers map[string]string, bodyBytes []byte) {
	var customHeaders map[string]string
	cfg, err := s.queries.GetForwardingConfigByEndpoint(ctx, job.EndpointID)
	if err == nil {
		customHeaders = s.resolveCustomHeaders(cfg)
	}

	fwdConfig := forwarding.Config{
		EndpointID: uuid.UUID(job.EndpointID.Bytes).String(),
		TargetURL:  job.TargetUrl,
		MaxRetries: 1, // Single direct attempt per outbox lease
		TimeoutMs:  5000,
		Headers:    customHeaders,
		Enabled:    true,
	}

	result, fwdErr := s.forwarder.ForwardRequest(ctx, fwdConfig, method, headers, bodyBytes)
	if fwdErr == nil && result != nil && result.Success {
		// SENT state
		_, _ = s.queries.CompleteOutboxJob(ctx, job.ID)
		_ = s.queries.UpdateRequestProcessingStatus(ctx, database.UpdateRequestProcessingStatusParams{
			ID:               job.RequestID,
			ProcessingStatus: "FORWARDED",
		})
		return
	}

	// Handle Failure with State Machine: RETRY_WAIT vs DLQ (#9)
	errMsg := "forwarding failed"
	if result != nil && result.ErrorMessage != "" {
		errMsg = result.ErrorMessage
	} else if fwdErr != nil {
		errMsg = fwdErr.Error()
	}

	nextAttempt := job.Attempts + 1
	if nextAttempt >= job.MaxRetries {
		// Max retries reached -> DLQ
		_, _ = s.queries.FailOutboxJob(ctx, database.FailOutboxJobParams{
			ID:          job.ID,
			Status:      "DLQ",
			LastError:   pgtype.Text{String: errMsg, Valid: true},
			NextRetryAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
		})
		_ = s.queries.UpdateRequestProcessingStatus(ctx, database.UpdateRequestProcessingStatusParams{
			ID:               job.RequestID,
			ProcessingStatus: "DLQ_FAILED",
		})
	} else {
		// Calculate exponential backoff: 2^attempt seconds (e.g., 2s, 4s, 8s)
		backoffSec := math.Pow(2, float64(nextAttempt))
		nextRetry := time.Now().Add(time.Duration(backoffSec) * time.Second)

		_, _ = s.queries.FailOutboxJob(ctx, database.FailOutboxJobParams{
			ID:          job.ID,
			Status:      "RETRY_WAIT",
			LastError:   pgtype.Text{String: errMsg, Valid: true},
			NextRetryAt: pgtype.Timestamptz{Time: nextRetry, Valid: true},
		})
	}
}

func (s *ForwardingService) ListDLQ(ctx context.Context, endpointID string) ([]database.ForwardingDlq, error) {
	epUUID, err := uuid.Parse(endpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}

	return s.queries.ListDLQRecordsByEndpoint(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
}

// RetryDLQRecord retries a failed DLQ record with atomic lease lock (#2.2, #9)
func (s *ForwardingService) RetryDLQRecord(ctx context.Context, dlqID string) error {
	dlqUUID, err := uuid.Parse(dlqID)
	if err != nil {
		return fmt.Errorf("invalid DLQ ID: %w", err)
	}

	record, err := s.queries.GetDLQRecordByID(ctx, pgtype.UUID{Bytes: dlqUUID, Valid: true})
	if err != nil {
		return fmt.Errorf("DLQ record not found: %w", err)
	}

	// Retrieve original HTTP method
	httpMethod := "POST"
	origReq, origErr := s.queries.GetCapturedRequestByID(ctx, record.RequestID)
	if origErr == nil {
		httpMethod = origReq.HttpMethod
	}

	var customHeaders map[string]string
	cfg, err := s.queries.GetForwardingConfigByEndpoint(ctx, record.EndpointID)
	if err == nil {
		customHeaders = s.resolveCustomHeaders(cfg)
	}

	targetURL := record.TargetUrl
	if cfg.TargetUrl != "" {
		targetURL = cfg.TargetUrl
	}

	fwdConfig := forwarding.Config{
		EndpointID: uuid.UUID(record.EndpointID.Bytes).String(),
		TargetURL:  targetURL,
		MaxRetries: 1,
		TimeoutMs:  5000,
		Headers:    customHeaders,
		Enabled:    true,
	}

	bodyBytes := []byte(record.Payload.String)
	result, fwdErr := s.forwarder.ForwardRequest(ctx, fwdConfig, httpMethod, customHeaders, bodyBytes)

	if fwdErr == nil && result != nil && result.Success {
		_, _ = s.queries.CompleteOutboxJob(ctx, record.ID)
		_ = s.queries.UpdateRequestProcessingStatus(ctx, database.UpdateRequestProcessingStatusParams{
			ID:               record.RequestID,
			ProcessingStatus: "FORWARDED",
		})
		return nil
	}

	// Still failed -> keep in DLQ
	errMsg := "retry failed"
	if result != nil && result.ErrorMessage != "" {
		errMsg = result.ErrorMessage
	} else if fwdErr != nil {
		errMsg = fwdErr.Error()
	}

	_, _ = s.queries.FailOutboxJob(ctx, database.FailOutboxJobParams{
		ID:          record.ID,
		Status:      "DLQ",
		LastError:   pgtype.Text{String: errMsg, Valid: true},
		NextRetryAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
	})

	return fmt.Errorf("retry failed: %s", errMsg)
}

func (s *ForwardingService) PurgeDLQ(ctx context.Context, endpointID string) error {
	epUUID, err := uuid.Parse(endpointID)
	if err != nil {
		return fmt.Errorf("invalid endpoint ID: %w", err)
	}

	return s.queries.DeleteDLQRecordsByEndpoint(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
}

// resolveCustomHeaders decrypts stored headers if encrypted, or unmarshals JSON directly
func (s *ForwardingService) resolveCustomHeaders(cfg database.ForwardingConfig) map[string]string {
	if len(cfg.CustomHeaders) == 0 {
		return nil
	}

	// 1. Check for encrypted JSON envelope {"_encrypted": "..."}
	var env map[string]string
	if err := json.Unmarshal(cfg.CustomHeaders, &env); err == nil {
		if encVal, ok := env["_encrypted"]; ok && encVal != "" {
			if s.encryptionKey != "" {
				decrypted, dErr := envelope.Decrypt(s.encryptionKey, encVal)
				if dErr == nil && decrypted != "" {
					var headers map[string]string
					if err := json.Unmarshal([]byte(decrypted), &headers); err == nil {
						return headers
					}
				}
			}
		} else {
			// Direct unencrypted map
			return env
		}
	}

	// 2. Direct string decrypt fallback
	if s.encryptionKey != "" {
		decrypted, err := envelope.Decrypt(s.encryptionKey, string(cfg.CustomHeaders))
		if err == nil && decrypted != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(decrypted), &headers); err == nil {
				return headers
			}
		}
	}

	var headers map[string]string
	_ = json.Unmarshal(cfg.CustomHeaders, &headers)
	return headers
}
