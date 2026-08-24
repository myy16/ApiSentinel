package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/forwarding"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type ForwardingService struct {
	queries   *database.Queries
	forwarder *forwarding.Forwarder
}

func NewForwardingService(queries *database.Queries) *ForwardingService {
	return &ForwardingService{
		queries:   queries,
		forwarder: forwarding.NewForwarder(),
	}
}

type SaveForwardingConfigInput struct {
	EndpointID    string            `json:"endpointId"`
	TargetURL     string            `json:"targetUrl"`
	MaxRetries    int               `json:"maxRetries"`
	TimeoutMs     int               `json:"timeoutMs"`
	CustomHeaders map[string]string `json:"customHeaders"`
	IsEnabled     bool              `json:"isEnabled"`
}

func (s *ForwardingService) SaveConfig(ctx context.Context, input SaveForwardingConfigInput) (*database.ForwardingConfig, error) {
	epUUID, err := uuid.Parse(input.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}

	headersJSON, _ := json.Marshal(input.CustomHeaders)
	maxRetries := input.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	timeoutMs := input.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	cfg, err := s.queries.UpsertForwardingConfig(ctx, database.UpsertForwardingConfigParams{
		EndpointID:    pgtype.UUID{Bytes: epUUID, Valid: true},
		TargetUrl:     input.TargetURL,
		MaxRetries:    int32(maxRetries),
		TimeoutMs:     int32(timeoutMs),
		CustomHeaders: headersJSON,
		IsEnabled:     input.IsEnabled,
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
	return &cfg, nil
}

// ForwardCleanWebhook handles async upstream forwarding and records DLQ on failure
func (s *ForwardingService) ForwardCleanWebhook(ctx context.Context, endpointID string, reqID string, method string, headers map[string]string, body []byte) {
	go func() {
		bgCtx := context.Background()
		epUUID, err := uuid.Parse(endpointID)
		if err != nil {
			return
		}

		cfg, err := s.queries.GetForwardingConfigByEndpoint(bgCtx, pgtype.UUID{Bytes: epUUID, Valid: true})
		if err != nil || !cfg.IsEnabled || cfg.TargetUrl == "" {
			return
		}

		var customHeaders map[string]string
		_ = json.Unmarshal(cfg.CustomHeaders, &customHeaders)

		fwdConfig := forwarding.Config{
			EndpointID: endpointID,
			TargetURL:  cfg.TargetUrl,
			MaxRetries: int(cfg.MaxRetries),
			TimeoutMs:  int(cfg.TimeoutMs),
			Headers:    customHeaders,
			Enabled:    cfg.IsEnabled,
		}

		result, err := s.forwarder.ForwardRequest(bgCtx, fwdConfig, method, headers, body)
		if err != nil || (result != nil && result.SavedToDLQ) {
			reqUUID, _ := uuid.Parse(reqID)
			errMsg := ""
			if result != nil {
				errMsg = result.ErrorMessage
			} else if err != nil {
				errMsg = err.Error()
			}

			_, dlqErr := s.queries.CreateDLQRecord(bgCtx, database.CreateDLQRecordParams{
				EndpointID: pgtype.UUID{Bytes: epUUID, Valid: true},
				RequestID:  pgtype.UUID{Bytes: reqUUID, Valid: true},
				TargetUrl:  cfg.TargetUrl,
				Attempts:   int32(fwdConfig.MaxRetries),
				LastError:  pgtype.Text{String: errMsg, Valid: true},
				Payload:    pgtype.Text{String: string(body), Valid: true},
				Status:     "FAILED",
			})
			if dlqErr != nil {
				log.Error().Err(dlqErr).Msg("Failed to record forwarding failure into DLQ")
			}
		}
	}()
}

func (s *ForwardingService) ListDLQ(ctx context.Context, endpointID string) ([]database.ForwardingDlq, error) {
	epUUID, err := uuid.Parse(endpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}

	return s.queries.ListDLQRecordsByEndpoint(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
}

