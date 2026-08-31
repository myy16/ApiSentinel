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
	ProjectID      string  `json:"projectId,omitempty"`
	ScanID         *string `json:"scanId,omitempty"`
	SourceType     string  `json:"sourceType"`
	ReqDisplayID   string  `json:"reqDisplayId"`
	EndpointName   string  `json:"endpointName"`
	Category       string  `json:"category"`
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	Action         string  `json:"action"`
	FieldPath      *string `json:"fieldPath"`
	FilePath       *string `json:"filePath,omitempty"`
	LineNumber     *int    `json:"lineNumber,omitempty"`
	CommitHash     *string `json:"commitHash,omitempty"`
	Repository     *string `json:"repository,omitempty"`
	Message        string  `json:"message"`
	EvidenceMasked *string `json:"evidenceMasked"`
	Confidence     float64 `json:"confidence"`
	CreatedAt      string  `json:"createdAt"`
}

type AgentScanResponse struct {
	ID            string `json:"id"`
	ProjectID     string `json:"projectId"`
	AgentID       string `json:"agentId"`
	Repository    string `json:"repository"`
	Branch        string `json:"branch"`
	CommitHash    string `json:"commitHash"`
	ScanType      string `json:"scanType"`
	TotalFindings int32  `json:"totalFindings"`
	Action        string `json:"action"`
	CreatedAt     string `json:"createdAt"`
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
		var filePath *string
		if r.FilePath.Valid {
			filePath = &r.FilePath.String
		}
		var lineNumber *int
		if r.LineNumber.Valid {
			lineVal := int(r.LineNumber.Int32)
			lineNumber = &lineVal
		}
		var commitHash *string
		if r.CommitHash.Valid {
			commitHash = &r.CommitHash.String
		}
		var repository *string
		if r.Repository.Valid {
			repository = &r.Repository.String
		}
		var scanID *string
		if r.ScanID.Valid {
			sID := uuid.UUID(r.ScanID.Bytes).String()
			scanID = &sID
		}
		var evidence *string
		if r.EvidenceMasked.Valid {
			evidence = &r.EvidenceMasked.String
		}

		var confidence float64 = 1.0
		if r.Confidence.Valid {
			confidence = r.Confidence.Float64
		}

		sourceType := r.SourceType
		if sourceType == "" {
			sourceType = "WEBHOOK"
		}

		res = append(res, FindingResponse{
			ID:             uuid.UUID(r.ID.Bytes).String(),
			RequestID:      uuid.UUID(r.RequestID.Bytes).String(),
			ProjectID:      uuid.UUID(r.ProjectID.Bytes).String(),
			ScanID:         scanID,
			SourceType:     sourceType,
			ReqDisplayID:   r.ReqDisplayID,
			EndpointName:   r.EndpointName,
			Category:       r.Category,
			Type:           r.Type,
			Severity:       r.Severity,
			Action:         r.Action,
			FieldPath:      fieldPath,
			FilePath:       filePath,
			LineNumber:     lineNumber,
			CommitHash:     commitHash,
			Repository:     repository,
			Message:        r.Message,
			EvidenceMasked: evidence,
			Confidence:     confidence,
			CreatedAt:      r.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}

func (s *FindingService) ListAgentScans(ctx context.Context, projectId string, limit, offset int32) ([]AgentScanResponse, error) {
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	scans, err := s.queries.ListAgentScansByProject(ctx, database.ListAgentScansByProjectParams{
		ProjectID: pgProjId,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	var res []AgentScanResponse
	for _, sc := range scans {
		res = append(res, AgentScanResponse{
			ID:            uuid.UUID(sc.ID.Bytes).String(),
			ProjectID:     uuid.UUID(sc.ProjectID.Bytes).String(),
			AgentID:       sc.AgentID,
			Repository:    sc.Repository,
			Branch:        sc.Branch,
			CommitHash:    sc.CommitHash,
			ScanType:      sc.ScanType,
			TotalFindings: sc.TotalFindings,
			Action:        sc.Action,
			CreatedAt:     sc.CreatedAt.Time.Format(time.RFC3339),
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
