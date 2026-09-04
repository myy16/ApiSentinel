package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/alerting"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/apisentinel/apisentinel/internal/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type AlertService struct {
	queries       *database.Queries
	dispatcher    *alerting.Dispatcher
	workerPool    *worker.Pool
	encryptionKey string
	digestAcc     *alerting.DeliveryDigestAccumulator
}

func NewAlertService(queries *database.Queries, workerPool *worker.Pool, encryptionKey ...string) *AlertService {
	encKey := ""
	if len(encryptionKey) > 0 {
		encKey = encryptionKey[0]
	}
	s := &AlertService{
		queries:       queries,
		dispatcher:    alerting.NewDispatcher(),
		workerPool:    workerPool,
		encryptionKey: encKey,
	}

	// Initialize digest accumulator: 2-minute window or 10 failures threshold
	s.digestAcc = alerting.NewDeliveryDigestAccumulator(2*time.Minute, 10, func(ctx context.Context, payload alerting.DeliveryAlertPayload) {
		s.dispatchDeliveryAlertToChannels(ctx, payload)
	})

	return s
}

// RecordDeliveryFailure routes critical non-retryable errors instantly, and aggregates transient 5xx into digests
func (s *AlertService) RecordDeliveryFailure(projectID, projectName, endpointID, endpointName, targetURL string, statusCode int, errMsg string, isDLQ bool) {
	// Instant Critical: 401 Unauthorized, 403 Forbidden, 404 Not Found, or DLQ final exhaustion
	if statusCode == 401 || statusCode == 403 || statusCode == 404 || isDLQ {
		epPrefix := endpointID
		if len(epPrefix) > 8 {
			epPrefix = epPrefix[:8]
		}
		payload := alerting.DeliveryAlertPayload{
			EventID:        fmt.Sprintf("crit-%s-%d", epPrefix, time.Now().UnixNano()),
			ProjectID:      projectID,
			ProjectName:    projectName,
			EndpointID:     endpointID,
			EndpointName:   endpointName,
			TargetURL:      targetURL,
			AlertKind:      "INSTANT_CRITICAL",
			StatusCode:     statusCode,
			ErrorType:      "DELIVERY_CRITICAL_HALT",
			ErrorMessage:   errMsg,
			TotalFailures:  1,
			WindowDuration: "instant",
			Timestamp:      time.Now().Format(time.RFC3339),
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			s.dispatchDeliveryAlertToChannels(ctx, payload)
		}()
		return
	}

	// Transient 5xx / Timeouts -> Accumulate in Anti-Spam Digest Window
	if s.digestAcc != nil {
		s.digestAcc.RecordFailure(projectID, projectName, endpointID, endpointName, targetURL, statusCode, errMsg)
	}
}

func (s *AlertService) dispatchDeliveryAlertToChannels(ctx context.Context, payload alerting.DeliveryAlertPayload) {
	projUUID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return
	}

	channels, err := s.queries.ListAlertChannelsByProject(ctx, pgtype.UUID{Bytes: projUUID, Valid: true})
	if err != nil || len(channels) == 0 {
		return
	}

	for _, ch := range channels {
		if !ch.IsEnabled {
			continue
		}
		targetURL := s.resolveChannelURL(ch)
		if err := s.dispatcher.DispatchDeliveryAlert(ctx, alerting.ChannelType(ch.ChannelType), targetURL, payload); err != nil {
			log.Warn().Err(err).Str("channel", ch.Name).Msg("Failed to dispatch delivery alert")
		}
	}
}

type CreateAlertChannelInput struct {
	ProjectID   string `json:"projectId"`
	Name        string `json:"name"`
	ChannelType string `json:"channelType"`
	WebhookURL  string `json:"webhookUrl"`
	MinSeverity string `json:"minSeverity"`
}

func (s *AlertService) CreateChannel(ctx context.Context, input CreateAlertChannelInput) (*database.AlertChannel, error) {
	projUUID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	minSev := input.MinSeverity
	if minSev == "" {
		minSev = "HIGH"
	}

	// Validate channel type (#22)
	normalizedType := strings.ToUpper(input.ChannelType)
	if !isValidChannelType(normalizedType) {
		return nil, fmt.Errorf("geçersiz kanal tipi: %s. Kabul edilen tipler: SLACK, DISCORD, TELEGRAM, WEBHOOK", input.ChannelType)
	}

	// Encrypt webhook URL before persisting to PostgreSQL (#3.1, #5)
	storedURL := input.WebhookURL
	if s.encryptionKey != "" {
		encrypted, err := envelope.Encrypt(s.encryptionKey, input.WebhookURL)
		if err != nil {
			return nil, fmt.Errorf("webhook URL could not be encrypted: %w", err)
		}
		if encrypted != "" {
			storedURL = encrypted
		}
	}

	channel, err := s.queries.CreateAlertChannel(ctx, database.CreateAlertChannelParams{
		ProjectID:   pgtype.UUID{Bytes: projUUID, Valid: true},
		Name:        input.Name,
		ChannelType: normalizedType,
		WebhookUrl:  storedURL,
		MinSeverity: minSev,
		IsEnabled:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create alert channel: %w", err)
	}

	// Mask the returned URL so secrets are never returned in response (#3.2)
	channel.WebhookUrl = envelope.MaskWebhookURL(input.WebhookURL)
	return &channel, nil
}

func (s *AlertService) ListChannels(ctx context.Context, projectID string) ([]database.AlertChannel, error) {
	projUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	channels, err := s.queries.ListAlertChannelsByProject(ctx, pgtype.UUID{Bytes: projUUID, Valid: true})
	if err != nil {
		return nil, err
	}

	// Mask Webhook URLs in listed channels
	for i := range channels {
		rawURL := s.resolveChannelURL(channels[i])
		channels[i].WebhookUrl = envelope.MaskWebhookURL(rawURL)
	}

	return channels, nil
}

func (s *AlertService) DeleteChannel(ctx context.Context, channelID string) error {
	chUUID, err := uuid.Parse(channelID)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %w", err)
	}

	return s.queries.DeleteAlertChannel(ctx, pgtype.UUID{Bytes: chUUID, Valid: true})
}

func (s *AlertService) SendTestAlert(ctx context.Context, channelID string) error {
	chUUID, err := uuid.Parse(channelID)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %w", err)
	}

	ch, err := s.queries.GetAlertChannelByID(ctx, pgtype.UUID{Bytes: chUUID, Valid: true})
	if err != nil {
		return fmt.Errorf("alert channel not found: %w", err)
	}

	targetURL := s.resolveChannelURL(ch)

	testPayload := alerting.AlertPayload{
		EventID:        uuid.New().String(),
		ProjectName:    "Production API Gateway",
		EndpointName:   "Stripe Webhook Ingestion",
		Category:       "SECRET_LEAK",
		FindingType:    "OPENAI_API_KEY",
		Severity:       "CRITICAL",
		PolicyAction:   "BLOCK",
		RequestID:      "req_test_" + uuid.New().String()[:8],
		EvidenceMasked: "sk-proj-****...****tQUA",
		Message:        "Test Bildirimi: ApiSentinel Gerçek Zamanlı Koruma Aktif!",
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	return s.dispatcher.Dispatch(ctx, alerting.ChannelType(ch.ChannelType), targetURL, testPayload)
}

// DispatchForFindings sends alerts in background to all active channels of the project
func (s *AlertService) DispatchForFindings(projectID string, projectName string, endpointName string, reqID string, findings []database.SecurityFinding, policyAction string) {
	task := func(taskCtx context.Context) {
		ctx, cancel := context.WithTimeout(taskCtx, 10*time.Second)
		defer cancel()

		projUUID, err := uuid.Parse(projectID)
		if err != nil {
			return
		}

		channels, err := s.queries.ListAlertChannelsByProject(ctx, pgtype.UUID{Bytes: projUUID, Valid: true})
		if err != nil || len(channels) == 0 {
			return
		}

		for _, f := range findings {
			payload := alerting.AlertPayload{
				EventID:        uuid.New().String(),
				ProjectName:    projectName,
				EndpointName:   endpointName,
				Category:       f.Category,
				FindingType:    f.Type,
				Severity:       f.Severity,
				PolicyAction:   policyAction,
				RequestID:      reqID,
				EvidenceMasked: f.EvidenceMasked.String,
				Message:        f.Message,
				Timestamp:      time.Now().Format(time.RFC3339),
			}

			for _, ch := range channels {
				if !ch.IsEnabled {
					continue
				}
				if severityOrder(f.Severity) < severityOrder(ch.MinSeverity) {
					continue
				}
				targetURL := s.resolveChannelURL(ch)
				if err := s.dispatcher.Dispatch(ctx, alerting.ChannelType(ch.ChannelType), targetURL, payload); err != nil {
					log.Warn().Err(err).Str("channel", ch.Name).Msg("Failed to dispatch alert")
				}
			}
		}
	}

	if s.workerPool != nil {
		if err := s.workerPool.Submit(task); err != nil {
			log.Warn().Err(err).Msg("Worker pool full or closed; alert dispatch was not scheduled")
		}
	} else {
		log.Warn().Msg("Worker pool unavailable; alert dispatch was not scheduled")
	}
}

// resolveChannelURL decrypts the stored webhook URL if encrypted, or falls back gracefully
func (s *AlertService) resolveChannelURL(ch database.AlertChannel) string {
	if s.encryptionKey != "" && ch.WebhookUrl != "" {
		decrypted, err := envelope.Decrypt(s.encryptionKey, ch.WebhookUrl)
		if err == nil && decrypted != "" {
			return decrypted
		}
	}
	return ch.WebhookUrl
}

// severityOrder returns a numeric ranking for severity comparison.
func severityOrder(severity string) int {
	switch strings.ToUpper(severity) {
	case "LOW":
		return 1
	case "MEDIUM":
		return 2
	case "HIGH":
		return 3
	case "CRITICAL":
		return 4
	default:
		return 0
	}
}

// isValidChannelType validates that the given channel type is supported.
func isValidChannelType(channelType string) bool {
	switch channelType {
	case "SLACK", "DISCORD", "TELEGRAM", "WEBHOOK":
		return true
	default:
		return false
	}
}
