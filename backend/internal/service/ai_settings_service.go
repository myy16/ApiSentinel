package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/apisentinel/apisentinel/internal/ai"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AISettingsService struct {
	queries   *database.Queries
	explainer *ai.Explainer
}

func NewAISettingsService(queries *database.Queries, explainer *ai.Explainer) *AISettingsService {
	return &AISettingsService{
		queries:   queries,
		explainer: explainer,
	}
}

type AISettingsResponse struct {
	OrganizationID        string   `json:"organizationId"`
	AIEnabled             bool     `json:"aiEnabled"`
	AIDataSharingLevel    string   `json:"aiDataSharingLevel"` // "NONE", "SANITIZED", "FULL_LOCAL"
	CustomRedactionKeys   []string `json:"customRedactionKeys"`
	SanitizationAvailable bool     `json:"sanitizationAvailable"`
}

type UpdateAISettingsInput struct {
	OrganizationID      string   `json:"organizationId"`
	AIEnabled           bool     `json:"aiEnabled"`
	AIDataSharingLevel  string   `json:"aiDataSharingLevel"`
	CustomRedactionKeys []string `json:"customRedactionKeys"`
}

func (s *AISettingsService) GetSettings(ctx context.Context, orgId string) (*AISettingsResponse, error) {
	orgUUID, err := uuid.Parse(orgId)
	if err != nil {
		return nil, errors.New("geçersiz organization ID")
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], orgUUID[:])
	pgOrgId.Valid = true

	org, err := s.queries.GetOrganizationAISettings(ctx, pgOrgId)
	if err != nil {
		return nil, errors.New("organizasyon bulunamadı")
	}

	var customKeys []string
	if len(org.AiCustomRedactionPatterns) > 0 {
		_ = json.Unmarshal(org.AiCustomRedactionPatterns, &customKeys)
	}
	if customKeys == nil {
		customKeys = []string{}
	}

	return &AISettingsResponse{
		OrganizationID:        uuid.UUID(org.ID.Bytes).String(),
		AIEnabled:             org.AiEnabled,
		AIDataSharingLevel:    org.AiDataSharingLevel,
		CustomRedactionKeys:   customKeys,
		SanitizationAvailable: true,
	}, nil
}

func (s *AISettingsService) UpdateSettings(ctx context.Context, input UpdateAISettingsInput) (*AISettingsResponse, error) {
	orgUUID, err := uuid.Parse(input.OrganizationID)
	if err != nil {
		return nil, errors.New("geçersiz organization ID")
	}

	var pgOrgId pgtype.UUID
	copy(pgOrgId.Bytes[:], orgUUID[:])
	pgOrgId.Valid = true

	level := input.AIDataSharingLevel
	if level != "NONE" && level != "SANITIZED" && level != "FULL_LOCAL" {
		level = "SANITIZED"
	}

	patternsBytes, err := json.Marshal(input.CustomRedactionKeys)
	if err != nil {
		patternsBytes = []byte("[]")
	}

	org, err := s.queries.UpdateOrganizationAISettings(ctx, database.UpdateOrganizationAISettingsParams{
		ID:                          pgOrgId,
		AiEnabled:                   input.AIEnabled,
		AiDataSharingLevel:          level,
		AiCustomRedactionPatterns:   patternsBytes,
	})
	if err != nil {
		return nil, err
	}

	var customKeys []string
	if len(org.AiCustomRedactionPatterns) > 0 {
		_ = json.Unmarshal(org.AiCustomRedactionPatterns, &customKeys)
	}
	if customKeys == nil {
		customKeys = []string{}
	}

	return &AISettingsResponse{
		OrganizationID:        uuid.UUID(org.ID.Bytes).String(),
		AIEnabled:             org.AiEnabled,
		AIDataSharingLevel:    org.AiDataSharingLevel,
		CustomRedactionKeys:   customKeys,
		SanitizationAvailable: true,
	}, nil
}

type TestSanitizeInput struct {
	SampleText       string   `json:"sampleText"`
	CustomRedactKeys []string `json:"customRedactKeys"`
}

type TestSanitizeResponse struct {
	OriginalText   string            `json:"originalText"`
	SanitizedText  string            `json:"sanitizedText"`
	RedactionCount int               `json:"redactionCount"`
	MaskedTypes    []string          `json:"maskedTypes"`
	PromptSafety   ai.PromptSecurityCheck `json:"promptSafety"`
	Details        map[string]int    `json:"details"`
}

func (s *AISettingsService) TestSanitization(input TestSanitizeInput) *TestSanitizeResponse {
	san := ai.SanitizeForAI(input.SampleText, input.CustomRedactKeys)
	promptCheck := ai.InspectAndNeutralizePrompt(san.CleanText)

	return &TestSanitizeResponse{
		OriginalText:   input.SampleText,
		SanitizedText:  promptCheck.CleanedPrompt,
		RedactionCount: san.RedactionCount,
		MaskedTypes:    san.MaskedTypes,
		PromptSafety:   promptCheck,
		Details:        san.Details,
	}
}
