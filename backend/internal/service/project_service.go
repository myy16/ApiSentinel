package service

import (
	"context"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProjectService struct {
	queries *database.Queries
}

func NewProjectService(queries *database.Queries) *ProjectService {
	return &ProjectService{queries: queries}
}

type ProjectResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	CreatedAt      string `json:"createdAt"`
}

func (s *ProjectService) ListProjects(ctx context.Context, orgId string) ([]ProjectResponse, error) {
	parsedOrgId, err := uuid.Parse(orgId)
	if err != nil {
		return nil, err
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], parsedOrgId[:])
	pgOrgId.Valid = true

	projects, err := s.queries.ListProjectsByOrg(ctx, pgOrgId)
	if err != nil {
		return nil, err
	}

	var res []ProjectResponse
	for _, p := range projects {
		res = append(res, ProjectResponse{
			ID:             uuid.UUID(p.ID.Bytes).String(),
			OrganizationID: uuid.UUID(p.OrganizationID.Bytes).String(),
			Name:           p.Name,
			CreatedAt:      p.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}

func (s *ProjectService) CreateProject(ctx context.Context, orgId, name string) (*ProjectResponse, error) {
	parsedOrgId, err := uuid.Parse(orgId)
	if err != nil {
		return nil, err
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], parsedOrgId[:])
	pgOrgId.Valid = true

	p, err := s.queries.CreateProject(ctx, database.CreateProjectParams{
		OrganizationID: pgOrgId,
		Name:           name,
	})
	if err != nil {
		return nil, err
	}

	return &ProjectResponse{
		ID:             uuid.UUID(p.ID.Bytes).String(),
		OrganizationID: uuid.UUID(p.OrganizationID.Bytes).String(),
		Name:           p.Name,
		CreatedAt:      p.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *ProjectService) GetProject(ctx context.Context, orgId, projectId string) (*ProjectResponse, error) {
	parsedOrgId, err := uuid.Parse(orgId)
	if err != nil {
		return nil, err
	}
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], parsedOrgId[:])
	pgOrgId.Valid = true

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	p, err := s.queries.GetProjectByID(ctx, database.GetProjectByIDParams{
		ID:             pgProjId,
		OrganizationID: pgOrgId,
	})
	if err != nil {
		return nil, err
	}

	return &ProjectResponse{
		ID:             uuid.UUID(p.ID.Bytes).String(),
		OrganizationID: uuid.UUID(p.OrganizationID.Bytes).String(),
		Name:           p.Name,
		CreatedAt:      p.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, orgId, projectId, name string) (*ProjectResponse, error) {
	parsedOrgId, err := uuid.Parse(orgId)
	if err != nil {
		return nil, err
	}
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], parsedOrgId[:])
	pgOrgId.Valid = true

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	p, err := s.queries.UpdateProject(ctx, database.UpdateProjectParams{
		ID:             pgProjId,
		OrganizationID: pgOrgId,
		Name:           name,
	})
	if err != nil {
		return nil, err
	}

	return &ProjectResponse{
		ID:             uuid.UUID(p.ID.Bytes).String(),
		OrganizationID: uuid.UUID(p.OrganizationID.Bytes).String(),
		Name:           p.Name,
		CreatedAt:      p.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, orgId, projectId string) error {
	parsedOrgId, err := uuid.Parse(orgId)
	if err != nil {
		return err
	}
	parsedProjId, err := uuid.Parse(projectId)
	if err != nil {
		return err
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], parsedOrgId[:])
	pgOrgId.Valid = true

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], parsedProjId[:])
	pgProjId.Valid = true

	return s.queries.DeleteProject(ctx, database.DeleteProjectParams{
		ID:             pgProjId,
		OrganizationID: pgOrgId,
	})
}
