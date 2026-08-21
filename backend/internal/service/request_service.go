package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RequestService struct {
	queries *database.Queries
}

func NewRequestService(queries *database.Queries) *RequestService {
	return &RequestService{queries: queries}
}

type CapturedRequestResponse struct {
	ID               string                 `json:"id"`
	EndpointID       string                 `json:"endpointId"`
	RequestID        string                 `json:"requestId"`
	HTTPMethod       string                 `json:"httpMethod"`
	Headers          map[string]interface{} `json:"headers"`
	QueryParams      map[string]interface{} `json:"queryParams"`
	RawBody          *string                `json:"rawBody"`
	MaskedBody       *string                `json:"maskedBody"`
	ParsedJSON       interface{}            `json:"parsedJson"`
	ClientIP         *string                `json:"clientIp"`
	ResponseStatus   int32                  `json:"responseStatus"`
	ProcessingStatus string                 `json:"processingStatus"`
	CreatedAt        string                 `json:"createdAt"`
	Endpoint         *EndpointMeta          `json:"endpoint,omitempty"`
}

type EndpointMeta struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *RequestService) ListByProject(ctx context.Context, projectId string, limit, offset int32) ([]CapturedRequestResponse, error) {
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	rows, err := s.queries.ListRequestsByProject(ctx, database.ListRequestsByProjectParams{
		ProjectID: pgProjId,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	var res []CapturedRequestResponse
	for _, r := range rows {
		var headers map[string]interface{}
		json.Unmarshal(r.Headers, &headers)

		var query map[string]interface{}
		json.Unmarshal(r.QueryParams, &query)

		var raw *string
		if r.RawBody.Valid {
			raw = &r.RawBody.String
		}

		var masked *string
		if r.MaskedBody.Valid {
			masked = &r.MaskedBody.String
		}

		var parsed interface{}
		if len(r.ParsedJson) > 0 {
			json.Unmarshal(r.ParsedJson, &parsed)
		}

		var status int32 = 200
		if r.ResponseStatus.Valid {
			status = r.ResponseStatus.Int32
		}

		res = append(res, CapturedRequestResponse{
			ID:               uuid.UUID(r.ID.Bytes).String(),
			EndpointID:       uuid.UUID(r.EndpointID.Bytes).String(),
			RequestID:        r.RequestID,
			HTTPMethod:       r.HttpMethod,
			Headers:          headers,
			QueryParams:      query,
			RawBody:          raw,
			MaskedBody:       masked,
			ParsedJSON:       parsed,
			ResponseStatus:   status,
			ProcessingStatus: r.ProcessingStatus,
			CreatedAt:        r.CreatedAt.Time.Format(time.RFC3339),
			Endpoint: &EndpointMeta{
				Name: r.EndpointName,
				Slug: r.EndpointSlug,
			},
		})
	}

	return res, nil
}
