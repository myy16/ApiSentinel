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
