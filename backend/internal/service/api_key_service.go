package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	APIKeyPrefixLive = "apisent_live_"
	APIKeyPrefixTest = "apisent_test_"
)

var (
	ErrInvalidAPIKey = errors.New("invalid or revoked API key")
	ErrKeyNotFound   = errors.New("API key not found")
)

type APIKeyService struct {
	queries *database.Queries
}

func NewAPIKeyService(queries *database.Queries) *APIKeyService {
	return &APIKeyService{queries: queries}
}

type CreatedAPIKeyResponse struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"projectId"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"keyPrefix"`
	SecretKey string     `json:"secretKey"` // Full plaintext key returned ONLY upon creation
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type APIKeyItemResponse struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"projectId"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"keyPrefix"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	IsRevoked  bool       `json:"isRevoked"`
}

// GenerateAPIKey creates a new cryptographic API key and saves its SHA-256 hash to PostgreSQL.
func (s *APIKeyService) GenerateAPIKey(ctx context.Context, projectID, name, createdBy string, isLive bool, expiresAt *time.Time) (*CreatedAPIKeyResponse, error) {
	projUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var pgProjUUID pgtype.UUID
	copy(pgProjUUID.Bytes[:], projUUID[:])
	pgProjUUID.Valid = true

	var pgCreatedBy pgtype.UUID
	if createdBy != "" {
		if uUUID, uErr := uuid.Parse(createdBy); uErr == nil {
			copy(pgCreatedBy.Bytes[:], uUUID[:])
			pgCreatedBy.Valid = true
		}
	}

	prefix := APIKeyPrefixLive
	if !isLive {
		prefix = APIKeyPrefixTest
	}

	// Generate 32 random bytes (64 hex characters)
	randomBytes := make([]byte, 24)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomSecret := hex.EncodeToString(randomBytes)
	fullKey := prefix + randomSecret

	// Compute SHA-256 hash of the full key
	hash := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(hash[:])

	var pgExpires pgtype.Timestamptz
	if expiresAt != nil {
		pgExpires = pgtype.Timestamptz{Time: *expiresAt, Valid: true}
	}

	created, err := s.queries.CreateAPIKey(ctx, database.CreateAPIKeyParams{
		ProjectID: pgProjUUID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   keyHash,
		CreatedBy: pgCreatedBy,
		ExpiresAt: pgExpires,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save API key to database: %w", err)
	}

	res := &CreatedAPIKeyResponse{
		ID:        uuid.UUID(created.ID.Bytes).String(),
		ProjectID: projectID,
		Name:      created.Name,
		KeyPrefix: created.KeyPrefix,
		SecretKey: fullKey,
		CreatedAt: created.CreatedAt.Time,
	}
	if created.ExpiresAt.Valid {
		res.ExpiresAt = &created.ExpiresAt.Time
	}

	return res, nil
}

// ValidateKey verifies an incoming API key string and updates last_used_at.
func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*database.ApiKey, error) {
	rawKey = strings.TrimSpace(rawKey)
	if !strings.HasPrefix(rawKey, APIKeyPrefixLive) && !strings.HasPrefix(rawKey, APIKeyPrefixTest) {
		return nil, ErrInvalidAPIKey
	}

	var prefix string
	if strings.HasPrefix(rawKey, APIKeyPrefixLive) {
		prefix = APIKeyPrefixLive
	} else {
		prefix = APIKeyPrefixTest
	}

	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	key, err := s.queries.GetAPIKeyByPrefixAndHash(ctx, database.GetAPIKeyByPrefixAndHashParams{
		KeyPrefix: prefix,
		KeyHash:   keyHash,
	})
	if err != nil {
		return nil, ErrInvalidAPIKey
	}

	// Check expiration
	if key.ExpiresAt.Valid && key.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrInvalidAPIKey
	}

	// Update last_used_at asynchronously
	go func() {
		_ = s.queries.UpdateAPIKeyLastUsed(context.Background(), key.ID)
	}()

	return &key, nil
}

// ListByProject returns all API keys for a project.
func (s *APIKeyService) ListByProject(ctx context.Context, projectID string) ([]APIKeyItemResponse, error) {
	projUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var pgProjUUID pgtype.UUID
	copy(pgProjUUID.Bytes[:], projUUID[:])
	pgProjUUID.Valid = true

	rows, err := s.queries.ListAPIKeysByProject(ctx, pgProjUUID)
	if err != nil {
		return nil, err
	}

	var items []APIKeyItemResponse
	for _, r := range rows {
		item := APIKeyItemResponse{
			ID:        uuid.UUID(r.ID.Bytes).String(),
			ProjectID: projectID,
			Name:      r.Name,
			KeyPrefix: r.KeyPrefix,
			CreatedAt: r.CreatedAt.Time,
			IsRevoked: r.RevokedAt.Valid,
		}
		if r.LastUsedAt.Valid {
			item.LastUsedAt = &r.LastUsedAt.Time
		}
		if r.ExpiresAt.Valid {
			item.ExpiresAt = &r.ExpiresAt.Time
		}
		if r.RevokedAt.Valid {
			item.RevokedAt = &r.RevokedAt.Time
		}
		items = append(items, item)
	}

	return items, nil
}

// RevokeKey immediately revokes an API key.
func (s *APIKeyService) RevokeKey(ctx context.Context, projectID, keyID string) error {
	pUUID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	kUUID, err := uuid.Parse(keyID)
	if err != nil {
		return err
	}

	var pgProjUUID, pgKeyUUID pgtype.UUID
	copy(pgProjUUID.Bytes[:], pUUID[:])
	pgProjUUID.Valid = true
	copy(pgKeyUUID.Bytes[:], kUUID[:])
	pgKeyUUID.Valid = true

	_, err = s.queries.RevokeAPIKey(ctx, database.RevokeAPIKeyParams{
		ID:        pgKeyUUID,
		ProjectID: pgProjUUID,
	})
	return err
}
