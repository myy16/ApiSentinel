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
	ID           string  `json:"id"`
	ProjectID    string  `json:"projectId"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Mode         string  `json:"mode"`
	IsActive     bool    `json:"isActive"`
	UpstreamURL  *string `json:"upstreamUrl"`
	RequestCount int64   `json:"requestCount"`
	CreatedAt    string  `json:"createdAt"`
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
			ID:           uuid.UUID(ep.ID.Bytes).String(),
			ProjectID:    uuid.UUID(ep.ProjectID.Bytes).String(),
			Slug:         ep.Slug,
			Name:         ep.Name,
			Mode:         ep.Mode,
			IsActive:     ep.IsActive,
			UpstreamURL:  upstream,
			RequestCount: ep.RequestCount,
			CreatedAt:    ep.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}

func (s *EndpointService) CreateEndpoint(ctx context.Context, projectId, name, slug, mode string, upstreamUrl *string) (*EndpointResponse, error) {
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	if slug == "" {
		b := make([]byte, 3)
		rand.Read(b)
		slug = strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + hex.EncodeToString(b)
	}

	if mode == "" {
		mode = "PASS"
	}

	var pgUpstream pgtype.Text
	if upstreamUrl != nil && *upstreamUrl != "" {
		pgUpstream = pgtype.Text{String: *upstreamUrl, Valid: true}
	}

	// Check if slug exists
	_, err = s.queries.GetEndpointBySlug(ctx, slug)
	if err == nil {
		return nil, errors.New("bu slug zaten kullanımda")
	}

	ep, err := s.queries.CreateEndpoint(ctx, database.CreateEndpointParams{
		ProjectID:   pgProjId,
		Slug:        slug,
		Name:        name,
		Mode:        mode,
		UpstreamUrl: pgUpstream,
		IsActive:    true,
	})
	if err != nil {
		return nil, err
	}

	var upstream *string
	if ep.UpstreamUrl.Valid {
		upstream = &ep.UpstreamUrl.String
	}

	return &EndpointResponse{
		ID:           uuid.UUID(ep.ID.Bytes).String(),
		ProjectID:    uuid.UUID(ep.ProjectID.Bytes).String(),
		Slug:         ep.Slug,
		Name:         ep.Name,
		Mode:         ep.Mode,
		IsActive:     ep.IsActive,
		UpstreamURL:  upstream,
		RequestCount: 0,
		CreatedAt:    ep.CreatedAt.Time.Format(time.RFC3339),
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

func (s *EndpointService) UpdateEndpoint(ctx context.Context, endpointId, projectId, name, mode string, isActive *bool, upstreamUrl *string) (*EndpointResponse, error) {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return nil, errors.New("geçersiz endpoint ID")
	}
	projUUID, err := uuid.Parse(projectId)
	if err != nil {
		return nil, errors.New("geçersiz project ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], projUUID[:])
	pgProjId.Valid = true

	var pgUpstream pgtype.Text
	if upstreamUrl != nil {
		pgUpstream = pgtype.Text{String: *upstreamUrl, Valid: true}
	}

	activeVal := true
	if isActive != nil {
		activeVal = *isActive
	}

	ep, err := s.queries.UpdateEndpoint(ctx, database.UpdateEndpointParams{
		ID:          pgEpId,
		ProjectID:   pgProjId,
		Name:        name,
		Mode:        mode,
		IsActive:    activeVal,
		UpstreamUrl: pgUpstream,
	})
	if err != nil {
		return nil, err
	}

	var upstream *string
	if ep.UpstreamUrl.Valid {
		upstream = &ep.UpstreamUrl.String
	}

	return &EndpointResponse{
		ID:          uuid.UUID(ep.ID.Bytes).String(),
		ProjectID:   uuid.UUID(ep.ProjectID.Bytes).String(),
		Slug:        ep.Slug,
		Name:        ep.Name,
		Mode:        ep.Mode,
		IsActive:    ep.IsActive,
		UpstreamURL: upstream,
		CreatedAt:   ep.CreatedAt.Time.Format(time.RFC3339),
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


