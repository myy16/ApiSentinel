package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type MockService struct {
	queries *database.Queries
}

func NewMockService(queries *database.Queries) *MockService {
	return &MockService{queries: queries}
}

type CreateMockRuleInput struct {
	Name            string                 `json:"name"`
	StatusCode      int                    `json:"statusCode"`
	DelayMs         int                    `json:"delayMs"`
	ResponseHeaders map[string]string      `json:"responseHeaders"`
	ResponseBody    map[string]interface{} `json:"responseBody"`
	Enabled         bool                   `json:"enabled"`
}

type MockRuleResponse struct {
	ID              string                 `json:"id"`
	EndpointID      string                 `json:"endpointId"`
	Name            string                 `json:"name"`
	StatusCode      int32                  `json:"statusCode"`
	DelayMs         int32                  `json:"delayMs"`
	ResponseHeaders map[string]string      `json:"responseHeaders"`
	ResponseBody    map[string]interface{} `json:"responseBody"`
	Enabled         bool                   `json:"enabled"`
}

func (s *MockService) CreateRule(ctx context.Context, endpointId string, input CreateMockRuleInput) (*MockRuleResponse, error) {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return nil, errors.New("geçersiz endpoint ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	headersBytes, _ := json.Marshal(input.ResponseHeaders)
	bodyBytes, _ := json.Marshal(input.ResponseBody)

	if input.StatusCode == 0 {
		input.StatusCode = 200
	}

	rule, err := s.queries.CreateMockRule(ctx, database.CreateMockRuleParams{
		EndpointID:      pgEpId,
		Name:            input.Name,
		StatusCode:      int32(input.StatusCode),
		DelayMs:         int32(input.DelayMs),
		ResponseHeaders: headersBytes,
		ResponseBody:    bodyBytes,
		Enabled:         input.Enabled,
	})
	if err != nil {
		return nil, err
	}

	return &MockRuleResponse{
		ID:              uuid.UUID(rule.ID.Bytes).String(),
		EndpointID:      endpointId,
		Name:            rule.Name,
		StatusCode:      rule.StatusCode,
		DelayMs:         rule.DelayMs,
		ResponseHeaders: input.ResponseHeaders,
		ResponseBody:    input.ResponseBody,
		Enabled:         rule.Enabled,
	}, nil
}

func (s *MockService) ListRules(ctx context.Context, endpointId string) ([]MockRuleResponse, error) {
	epUUID, err := uuid.Parse(endpointId)
	if err != nil {
		return nil, errors.New("geçersiz endpoint ID")
	}

	var pgEpId pgtype.UUID
	copy(pgEpId.Bytes[:], epUUID[:])
	pgEpId.Valid = true

	rules, err := s.queries.ListMockRulesByEndpoint(ctx, pgEpId)
	if err != nil {
		return nil, err
	}

	var res []MockRuleResponse
	for _, r := range rules {
		var headers map[string]string
		json.Unmarshal(r.ResponseHeaders, &headers)

		var body map[string]interface{}
		json.Unmarshal(r.ResponseBody, &body)

		res = append(res, MockRuleResponse{
			ID:              uuid.UUID(r.ID.Bytes).String(),
			EndpointID:      endpointId,
			Name:            r.Name,
			StatusCode:      r.StatusCode,
			DelayMs:         r.DelayMs,
			ResponseHeaders: headers,
			ResponseBody:    body,
			Enabled:         r.Enabled,
		})
	}

	return res, nil
}
