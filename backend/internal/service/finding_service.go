package service

import (
	"context"
	"fmt"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type FindingService struct {
	queries *database.Queries
}

func NewFindingService(queries *database.Queries) *FindingService {
	return &FindingService{queries: queries}
}

type FindingResponse struct {
	ID             string  `json:"id"`
	RequestID      string  `json:"requestId"`
	ReqDisplayID   string  `json:"reqDisplayId"`
	EndpointName   string  `json:"endpointName"`
	Category       string  `json:"category"`
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	Action         string  `json:"action"`
	FieldPath      *string `json:"fieldPath"`
	Message        string  `json:"message"`
	EvidenceMasked *string `json:"evidenceMasked"`
	Confidence     float64 `json:"confidence"`
	CreatedAt      string  `json:"createdAt"`
}

type FindingStatsResponse struct {
	CriticalCount int64 `json:"criticalCount"`
	HighCount     int64 `json:"highCount"`
	MediumCount   int64 `json:"mediumCount"`
	InfoCount     int64 `json:"infoCount"`
	TotalCount    int64 `json:"totalCount"`
}

func (s *FindingService) ListByProject(ctx context.Context, projectId string, limit, offset int32) ([]FindingResponse, error) {
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	rows, err := s.queries.ListFindingsByProject(ctx, database.ListFindingsByProjectParams{
		ProjectID: pgProjId,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	var res []FindingResponse
	for _, r := range rows {
		var fieldPath *string
		if r.FieldPath.Valid {
			fieldPath = &r.FieldPath.String
		}
		var evidence *string
		if r.EvidenceMasked.Valid {
			evidence = &r.EvidenceMasked.String
		}

		var confidence float64 = 1.0
		if r.Confidence.Valid {
			confidence = r.Confidence.Float64
		}

		res = append(res, FindingResponse{
			ID:             uuid.UUID(r.ID.Bytes).String(),
			RequestID:      uuid.UUID(r.RequestID.Bytes).String(),
			ReqDisplayID:   r.ReqDisplayID,
			EndpointName:   r.EndpointName,
			Category:       r.Category,
			Type:           r.Type,
			Severity:       r.Severity,
			Action:         r.Action,
			FieldPath:      fieldPath,
			Message:        r.Message,
			EvidenceMasked: evidence,
			Confidence:     confidence,
			CreatedAt:      r.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}

func (s *FindingService) GetStats(ctx context.Context, projectId string) (*FindingStatsResponse, error) {
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	stats, err := s.queries.GetFindingStats(ctx, pgProjId)
	if err != nil {
		return nil, err
	}

	return &FindingStatsResponse{
		CriticalCount: stats.CriticalCount,
		HighCount:     stats.HighCount,
		MediumCount:   stats.MediumCount,
		InfoCount:     stats.InfoCount,
		TotalCount:    stats.TotalCount,
	}, nil
}
