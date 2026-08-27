package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type WebhookSecurityService struct {
	queries       *database.Queries
	encryptionKey string
}

type SaveWebhookSecurityInput struct {
	Provider         string `json:"provider"`
	Secret           string `json:"secret"`
	RequireSignature bool   `json:"requireSignature"`
}

type WebhookSecurityResponse struct {
	EndpointID       string `json:"endpointId"`
	Provider         string `json:"provider"`
	RequireSignature bool   `json:"requireSignature"`
	Configured       bool   `json:"configured"`
	UpdatedAt        string `json:"updatedAt"`
}

func NewWebhookSecurityService(queries *database.Queries, encryptionKey string) *WebhookSecurityService {
	return &WebhookSecurityService{queries: queries, encryptionKey: encryptionKey}
}

func (s *WebhookSecurityService) Save(ctx context.Context, endpointID string, input SaveWebhookSecurityInput) (*WebhookSecurityResponse, error) {
	id, err := uuid.Parse(endpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	if provider != "stripe" && provider != "github" && provider != "shopify" && provider != "generic" {
		return nil, fmt.Errorf("unsupported webhook provider")
	}
	if strings.TrimSpace(input.Secret) == "" {
		return nil, fmt.Errorf("webhook signing secret is required")
	}
	encrypted, err := envelope.Encrypt(s.encryptionKey, input.Secret)
	if err != nil {
		return nil, err
	}
	record, err := s.queries.UpsertEndpointWebhookSecurity(ctx, database.UpsertEndpointWebhookSecurityParams{
		EndpointID:       pgtype.UUID{Bytes: id, Valid: true},
		Provider:         provider,
		EncryptedSecret:  encrypted,
		RequireSignature: input.RequireSignature,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save webhook security configuration: %w", err)
	}
	return webhookSecurityResponse(record), nil
}

func (s *WebhookSecurityService) Get(ctx context.Context, endpointID string) (*WebhookSecurityResponse, error) {
	id, err := uuid.Parse(endpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}
	record, err := s.queries.GetEndpointWebhookSecurity(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, err
	}
	return webhookSecurityResponse(record), nil
}

func (s *WebhookSecurityService) Delete(ctx context.Context, endpointID string) error {
	id, err := uuid.Parse(endpointID)
	if err != nil {
		return fmt.Errorf("invalid endpoint ID: %w", err)
	}
	return s.queries.DeleteEndpointWebhookSecurity(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func webhookSecurityResponse(record database.EndpointWebhookSecurity) *WebhookSecurityResponse {
	updatedAt := time.Time{}
	if record.UpdatedAt.Valid {
		updatedAt = record.UpdatedAt.Time
	}
	return &WebhookSecurityResponse{
		EndpointID:       uuid.UUID(record.EndpointID.Bytes).String(),
		Provider:         record.Provider,
		RequireSignature: record.RequireSignature,
		Configured:       true,
		UpdatedAt:        updatedAt.Format(time.RFC3339),
	}
}
