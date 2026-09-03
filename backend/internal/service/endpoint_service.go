package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type EndpointService struct {
	queries *database.Queries
}

func NewEndpointService(queries *database.Queries) *EndpointService {
	return &EndpointService{queries: queries}
}

type EndpointResponse struct {
	ID                  string  `json:"id"`
	ProjectID           string  `json:"projectId"`
	Slug                string  `json:"slug"`
	Name                string  `json:"name"`
	Mode                string  `json:"mode"`
	IsActive            bool    `json:"isActive"`
	UpstreamURL         *string `json:"upstreamUrl"`
	MaxPayloadSizeBytes int32   `json:"maxPayloadSizeBytes"`
	RateLimitRpm        int32   `json:"rateLimitRpm"`
	BurstThreshold      int32   `json:"burstThreshold"`
	RequestCount        int64   `json:"requestCount"`
	CreatedAt           string  `json:"createdAt"`
}

func (s *EndpointService) ListEndpoints(ctx context.Context, projectId string) ([]EndpointResponse, error) {
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	endpoints, err := s.queries.ListEndpointsByProject(ctx, pgProjId)
	if err != nil {
		return nil, err
	}

	var res []EndpointResponse
	for _, ep := range endpoints {
		var upstream *string
		if ep.UpstreamUrl.Valid {
			upstream = &ep.UpstreamUrl.String
		}

		res = append(res, EndpointResponse{
			ID:                  uuid.UUID(ep.ID.Bytes).String(),
			ProjectID:           uuid.UUID(ep.ProjectID.Bytes).String(),
			Slug:                ep.Slug,
			Name:                ep.Name,
			Mode:                ep.Mode,
			IsActive:            ep.IsActive,
			UpstreamURL:         upstream,
			MaxPayloadSizeBytes: ep.MaxPayloadSizeBytes,
			RateLimitRpm:        ep.RateLimitRpm,
			BurstThreshold:      ep.BurstThreshold,
			RequestCount:        ep.RequestCount,
			CreatedAt:           ep.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}

type CreateEndpointInput struct {
	ProjectID           string
	Name                string
	Slug                string
	Mode                string
	UpstreamURL         *string
	MaxPayloadSizeBytes int32
	RateLimitRpm        int32
	BurstThreshold      int32
}

func (s *EndpointService) CreateEndpoint(ctx context.Context, input CreateEndpointInput) (*EndpointResponse, error) {
	parsedProjId, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, err
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	slug := input.Slug
	if slug == "" {
		b := make([]byte, 3)
		rand.Read(b)
		slug = strings.ToLower(strings.ReplaceAll(input.Name, " ", "-")) + "-" + hex.EncodeToString(b)
	}

	mode := input.Mode
	if mode == "" {
		mode = "PASS"
	}

	maxPayload := input.MaxPayloadSizeBytes
	if maxPayload <= 0 {
		maxPayload = 5242880 // 5 MB
	}

	rateLimit := input.RateLimitRpm
	if rateLimit <= 0 {
		rateLimit = 120 // 120 RPM
	}

	burst := input.BurstThreshold
	if burst <= 0 {
		burst = 30 // 30 burst
	}

	var pgUpstream pgtype.Text
	if input.UpstreamURL != nil && *input.UpstreamURL != "" {
		pgUpstream = pgtype.Text{String: *input.UpstreamURL, Valid: true}
	}

	// Check if slug exists
	_, err = s.queries.GetEndpointBySlug(ctx, slug)
	if err == nil {
		return nil, errors.New("bu slug zaten kullanımda")
	}

	ep, err := s.queries.CreateEndpoint(ctx, database.CreateEndpointParams{
		ProjectID:           pgProjId,
		Slug:                slug,
		Name:                input.Name,
		Mode:                mode,
		UpstreamUrl:         pgUpstream,
		MaxPayloadSizeBytes: maxPayload,
		RateLimitRpm:        rateLimit,
		BurstThreshold:      burst,
		IsActive:            true,
	})
	if err != nil {
		return nil, err
	}

	var upstream *string
	if ep.UpstreamUrl.Valid {
		upstream = &ep.UpstreamUrl.String
	}

	return &EndpointResponse{
		ID:                  uuid.UUID(ep.ID.Bytes).String(),
		ProjectID:           uuid.UUID(ep.ProjectID.Bytes).String(),
		Slug:                ep.Slug,
		Name:                ep.Name,
		Mode:                ep.Mode,
		IsActive:            ep.IsActive,
		UpstreamURL:         upstream,
		MaxPayloadSizeBytes: ep.MaxPayloadSizeBytes,
		RateLimitRpm:        ep.RateLimitRpm,
		BurstThreshold:      ep.BurstThreshold,
		RequestCount:        0,
		CreatedAt:           ep.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *EndpointService) GetEndpointBySlug(ctx context.Context, slug string) (*database.Endpoint, error) {
	ep, err := s.queries.GetEndpointBySlug(ctx, slug)
	if err != nil {
		return nil, errors.New("endpoint bulunamadı")
	}
	return &ep, nil
}

func (s *EndpointService) SaveSchema(ctx context.Context, endpointId string, jsonSchemaBytes []byte) (*database.EndpointSchema, error) {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return nil, errors.New("geçersiz endpoint ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	res, err := s.queries.UpsertEndpointSchema(ctx, database.UpsertEndpointSchemaParams{
		EndpointID: pgEpId,
		JsonSchema: jsonSchemaBytes,
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *EndpointService) GetSchema(ctx context.Context, endpointId string) (*database.EndpointSchema, error) {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return nil, errors.New("geçersiz endpoint ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	res, err := s.queries.GetEndpointSchema(ctx, pgEpId)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *EndpointService) DeleteSchema(ctx context.Context, endpointId string) error {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return errors.New("geçersiz endpoint ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	return s.queries.DeleteEndpointSchema(ctx, pgEpId)
}

type UpdateEndpointInput struct {
	EndpointID          string
	ProjectID           string
	Name                string
	Mode                string
	IsActive            *bool
	UpstreamURL         *string
	MaxPayloadSizeBytes *int32
	RateLimitRpm        *int32
	BurstThreshold      *int32
}

func (s *EndpointService) UpdateEndpoint(ctx context.Context, input UpdateEndpointInput) (*EndpointResponse, error) {
	epUUID, err := uuid.Parse(input.EndpointID)
	if err != nil {
		return nil, errors.New("geçersiz endpoint ID")
	}
	projUUID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, errors.New("geçersiz project ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], projUUID[:])
	pgProjId.Valid = true

	existing, err := s.queries.GetEndpointByID(ctx, database.GetEndpointByIDParams{
		ID:        pgEpId,
		ProjectID: pgProjId,
	})
	if err != nil {
		return nil, errors.New("endpoint bulunamadı")
	}

	effectiveName := existing.Name
	if input.Name != "" {
		effectiveName = input.Name
	}

	effectiveMode := existing.Mode
	if input.Mode != "" {
		effectiveMode = input.Mode
	}

	effectiveActive := existing.IsActive
	if input.IsActive != nil {
		effectiveActive = *input.IsActive
	}

	var pgUpstream pgtype.Text
	if input.UpstreamURL != nil {
		pgUpstream = pgtype.Text{String: *input.UpstreamURL, Valid: true}
	} else if existing.UpstreamUrl.Valid {
		pgUpstream = existing.UpstreamUrl
	}

	effectiveMaxPayload := existing.MaxPayloadSizeBytes
	if input.MaxPayloadSizeBytes != nil && *input.MaxPayloadSizeBytes > 0 {
		effectiveMaxPayload = *input.MaxPayloadSizeBytes
	}

	effectiveRateLimit := existing.RateLimitRpm
	if input.RateLimitRpm != nil && *input.RateLimitRpm > 0 {
		effectiveRateLimit = *input.RateLimitRpm
	}

	effectiveBurst := existing.BurstThreshold
	if input.BurstThreshold != nil && *input.BurstThreshold > 0 {
		effectiveBurst = *input.BurstThreshold
	}

	ep, err := s.queries.UpdateEndpoint(ctx, database.UpdateEndpointParams{
		ID:                  pgEpId,
		ProjectID:           pgProjId,
		Name:                effectiveName,
		Mode:                effectiveMode,
		IsActive:            effectiveActive,
		UpstreamUrl:         pgUpstream,
		MaxPayloadSizeBytes: effectiveMaxPayload,
		RateLimitRpm:        effectiveRateLimit,
		BurstThreshold:      effectiveBurst,
	})
	if err != nil {
		return nil, err
	}

	var upstream *string
	if ep.UpstreamUrl.Valid {
		upstream = &ep.UpstreamUrl.String
	}

	return &EndpointResponse{
		ID:                  uuid.UUID(ep.ID.Bytes).String(),
		ProjectID:           uuid.UUID(ep.ProjectID.Bytes).String(),
		Slug:                ep.Slug,
		Name:                ep.Name,
		Mode:                ep.Mode,
		IsActive:            ep.IsActive,
		UpstreamURL:         upstream,
		MaxPayloadSizeBytes: ep.MaxPayloadSizeBytes,
		RateLimitRpm:        ep.RateLimitRpm,
		BurstThreshold:      ep.BurstThreshold,
		CreatedAt:           ep.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *EndpointService) DeleteEndpoint(ctx context.Context, endpointId, projectId string) error {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return errors.New("geçersiz endpoint ID")
	}
	projUUID, err := uuid.Parse(projectId)
	if err != nil {
		return errors.New("geçersiz project ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], projUUID[:])
	pgProjId.Valid = true

	return s.queries.DeleteEndpoint(ctx, database.DeleteEndpointParams{
		ID:        pgEpId,
		ProjectID: pgProjId,
	})
}
