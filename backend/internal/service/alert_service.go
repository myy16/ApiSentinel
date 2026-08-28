package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/alerting"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type AlertService struct {
	queries    *database.Queries
	dispatcher *alerting.Dispatcher
	workerPool *worker.Pool
}

func NewAlertService(queries *database.Queries, workerPool *worker.Pool) *AlertService {
	return &AlertService{
		queries:    queries,
		dispatcher: alerting.NewDispatcher(),
		workerPool: workerPool,
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

	channel, err := s.queries.CreateAlertChannel(ctx, database.CreateAlertChannelParams{
		ProjectID:   pgtype.UUID{Bytes: projUUID, Valid: true},
		Name:        input.Name,
		ChannelType: normalizedType,
		WebhookUrl:  input.WebhookURL,
		MinSeverity: minSev,
		IsEnabled:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create alert channel: %w", err)
	}

	return &channel, nil
}

func (s *AlertService) ListChannels(ctx context.Context, projectID string) ([]database.AlertChannel, error) {
	projUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	return s.queries.ListAlertChannelsByProject(ctx, pgtype.UUID{Bytes: projUUID, Valid: true})
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

	return s.dispatcher.Dispatch(ctx, alerting.ChannelType(ch.ChannelType), ch.WebhookUrl, testPayload)
}

// DispatchForFindings sends alerts in background to all active channels of the project
func (s *AlertService) DispatchForFindings(projectID string, projectName string, endpointName string, reqID string, findings []database.SecurityFinding, policyAction string) {
	task := func(taskCtx context.Context) {
		ctx, cancel := context.WithTimeout(taskCtx, 10*time.Second)
		defer cancel()

		channels, err := s.ListChannels(ctx, projectID)
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
				// Use each channel's own minSeverity instead of hardcoded filter (#21)
				if severityOrder(f.Severity) < severityOrder(ch.MinSeverity) {
					continue
				}
				if err := s.dispatcher.Dispatch(ctx, alerting.ChannelType(ch.ChannelType), ch.WebhookUrl, payload); err != nil {
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

// severityOrder returns a numeric ranking for severity comparison.
// Higher number = more severe. Unknown values default to 0.
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
