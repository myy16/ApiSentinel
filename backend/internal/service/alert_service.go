package service

import (
	"context"
	"fmt"
	"time"

	"github.com/apisentinel/apisentinel/internal/alerting"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type AlertService struct {
	queries    *database.Queries
	dispatcher *alerting.Dispatcher
}

func NewAlertService(queries *database.Queries) *AlertService {
	return &AlertService{
		queries:    queries,
		dispatcher: alerting.NewDispatcher(),
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

	channel, err := s.queries.CreateAlertChannel(ctx, database.CreateAlertChannelParams{
		ProjectID:   pgtype.UUID{Bytes: projUUID, Valid: true},
		Name:        input.Name,
		ChannelType: input.ChannelType,
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
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		channels, err := s.ListChannels(ctx, projectID)
		if err != nil || len(channels) == 0 {
			return
		}

		for _, f := range findings {
			if f.Severity != "CRITICAL" && f.Severity != "HIGH" {
				continue
			}

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
				if err := s.dispatcher.Dispatch(ctx, alerting.ChannelType(ch.ChannelType), ch.WebhookUrl, payload); err != nil {
					log.Warn().Err(err).Str("channel", ch.Name).Msg("Failed to dispatch alert")
				}
			}
		}
	}()
}
