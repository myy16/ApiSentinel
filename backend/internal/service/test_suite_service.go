package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type TestSuiteService struct {
	queries       *database.Queries
	replayService *ReplayService
	encryptionKey string
}

func NewTestSuiteService(queries *database.Queries, replayService *ReplayService, encryptionKey ...string) *TestSuiteService {
	key := ""
	if len(encryptionKey) > 0 {
		key = encryptionKey[0]
	}
	return &TestSuiteService{
		queries:       queries,
		replayService: replayService,
		encryptionKey: key,
	}
}

type CreateTestSuiteParams struct {
	ProjectID         string            `json:"projectId"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	RequestIDs        []string          `json:"requestIds"`
	TargetEnvironment string            `json:"targetEnvironment"`
	TargetURL         string            `json:"targetUrl"`
	RenewIdempotency  bool              `json:"renewIdempotency"`
	CustomHeaders     map[string]string `json:"customHeaders,omitempty"`
}

type TestSuiteStepResult struct {
	StepIndex      int               `json:"stepIndex"`
	RequestID      string            `json:"requestId"`
	TargetURL      string            `json:"targetUrl"`
	ResponseStatus int               `json:"responseStatus"`
	LatencyMs      int64             `json:"latencyMs"`
	Status         string            `json:"status"` // "PASSED", "FAILED"
	ErrorMessage   string            `json:"errorMessage,omitempty"`
	Replacements   map[string]string `json:"replacements,omitempty"`
}

type TestSuiteRunReport struct {
	RunID          string                `json:"runId"`
	SuiteID        string                `json:"suiteId"`
	SuiteName      string                `json:"suiteName"`
	Status         string                `json:"status"` // "PASSED", "FAILED", "PARTIAL"
	TotalSteps     int                   `json:"totalSteps"`
	PassedSteps    int                   `json:"passedSteps"`
	FailedSteps    int                   `json:"failedSteps"`
	TotalLatencyMs int                   `json:"totalLatencyMs"`
	StepResults    []TestSuiteStepResult `json:"stepResults"`
	CreatedAt      string                `json:"createdAt"`
	CompletedAt    string                `json:"completedAt"`
}

func (s *TestSuiteService) CreateSuite(ctx context.Context, params CreateTestSuiteParams) (*database.ReplayTestSuite, error) {
	if params.Name == "" {
		return nil, errors.New("test paketi adı zorunludur")
	}
	if len(params.RequestIDs) == 0 {
		return nil, errors.New("en az bir istek seçilmelidir")
	}

	projUUID, err := uuid.Parse(params.ProjectID)
	if err != nil {
		return nil, errors.New("geçersiz proje ID formatı")
	}

	targetProjPG := pgtype.UUID{Bytes: projUUID, Valid: true}

	// Verify all requestIDs belong to this project
	for _, reqIDStr := range params.RequestIDs {
		reqUUID, err := uuid.Parse(reqIDStr)
		if err != nil {
			return nil, fmt.Errorf("geçersiz istek ID formatı: %s", reqIDStr)
		}
		reqRecord, err := s.queries.GetCapturedRequestByID(ctx, pgtype.UUID{Bytes: reqUUID, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("istek bulunamadı: %s", reqIDStr)
		}
		if reqRecord.ProjectID != targetProjPG {
			return nil, fmt.Errorf("seçilen istek (%s) bu projeye ait değil", reqIDStr)
		}
	}

	reqIDsJSON, _ := json.Marshal(params.RequestIDs)
	customHeadersJSON, _ := json.Marshal(params.CustomHeaders)
	if s.encryptionKey != "" && len(params.CustomHeaders) > 0 {
		encVal, encErr := envelope.Encrypt(s.encryptionKey, string(customHeadersJSON))
		if encErr != nil {
			return nil, fmt.Errorf("custom headers could not be encrypted: %w", encErr)
		}
		if encVal != "" {
			envelopePayload, _ := json.Marshal(map[string]string{"_encrypted": encVal})
			customHeadersJSON = envelopePayload
		}
	}

	if params.TargetEnvironment == "" {
		params.TargetEnvironment = "STAGING"
	}

	suite, err := s.queries.CreateReplayTestSuite(ctx, database.CreateReplayTestSuiteParams{
		ProjectID:         targetProjPG,
		Name:              params.Name,
		Description:       pgtype.Text{String: params.Description, Valid: params.Description != ""},
		RequestIds:        reqIDsJSON,
		TargetEnvironment: params.TargetEnvironment,
		TargetUrl:         pgtype.Text{String: params.TargetURL, Valid: params.TargetURL != ""},
		RenewIdempotency:  params.RenewIdempotency,
		CustomHeaders:     customHeadersJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("test paketi kaydedilemedi: %w", err)
	}

	suite.CustomHeaders = s.maskedHeadersJSON(suite.CustomHeaders)
	return &suite, nil
}

func (s *TestSuiteService) ListSuites(ctx context.Context, projectID string) ([]database.ReplayTestSuite, error) {
	projUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, errors.New("geçersiz proje ID formatı")
	}

	suites, err := s.queries.ListReplayTestSuitesByProject(ctx, pgtype.UUID{Bytes: projUUID, Valid: true})
	if err != nil {
		return nil, err
	}
	for i := range suites {
		suites[i].CustomHeaders = s.maskedHeadersJSON(suites[i].CustomHeaders)
	}
	return suites, nil
}

func (s *TestSuiteService) GetSuite(ctx context.Context, suiteID string) (*database.ReplayTestSuite, []database.ReplayTestRun, error) {
	sUUID, err := uuid.Parse(suiteID)
	if err != nil {
		return nil, nil, errors.New("geçersiz test paketi ID formatı")
	}

	suite, err := s.queries.GetReplayTestSuiteByID(ctx, pgtype.UUID{Bytes: sUUID, Valid: true})
	if err != nil {
		return nil, nil, errors.New("test paketi bulunamadı")
	}

	runs, _ := s.queries.ListReplayTestRunsBySuite(ctx, pgtype.UUID{Bytes: sUUID, Valid: true})
	suite.CustomHeaders = s.maskedHeadersJSON(suite.CustomHeaders)

	return &suite, runs, nil
}

func (s *TestSuiteService) DeleteSuite(ctx context.Context, suiteID, projectID string) error {
	sUUID, err := uuid.Parse(suiteID)
	if err != nil {
		return errors.New("geçersiz test paketi ID formatı")
	}
	pUUID, err := uuid.Parse(projectID)
	if err != nil {
		return errors.New("geçersiz proje ID formatı")
	}

	return s.queries.DeleteReplayTestSuite(ctx, database.DeleteReplayTestSuiteParams{
		ID:        pgtype.UUID{Bytes: sUUID, Valid: true},
		ProjectID: pgtype.UUID{Bytes: pUUID, Valid: true},
	})
}

func (s *TestSuiteService) resolveCustomHeaders(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var envelopeMap map[string]string
	if err := json.Unmarshal(raw, &envelopeMap); err == nil {
		if encVal, ok := envelopeMap["_encrypted"]; ok && encVal != "" {
			if s.encryptionKey != "" {
				if dec, dErr := envelope.Decrypt(s.encryptionKey, encVal); dErr == nil && dec != "" {
					var origHeaders map[string]string
					if jErr := json.Unmarshal([]byte(dec), &origHeaders); jErr == nil {
						return origHeaders
					}
				}
			}
			return nil
		}
	}
	if s.encryptionKey != "" {
		if dec, err := envelope.Decrypt(s.encryptionKey, string(raw)); err == nil && dec != "" {
			var origHeaders map[string]string
			if jErr := json.Unmarshal([]byte(dec), &origHeaders); jErr == nil {
				return origHeaders
			}
		}
	}
	var plainHeaders map[string]string
	if err := json.Unmarshal(raw, &plainHeaders); err == nil {
		return plainHeaders
	}
	return nil
}

func (s *TestSuiteService) maskedHeadersJSON(raw []byte) []byte {
	headers := s.resolveCustomHeaders(raw)
	if len(headers) == 0 {
		return []byte("{}")
	}
	masked := make(map[string]string, len(headers))
	for k, v := range headers {
		masked[k] = envelope.MaskHeaderValue(k, v)
	}
	maskedJSON, err := json.Marshal(masked)
	if err != nil {
		return []byte("{}")
	}
	return maskedJSON
}

func (s *TestSuiteService) RunSuite(ctx context.Context, suiteID, userID, clientIP string) (*TestSuiteRunReport, error) {
	sUUID, err := uuid.Parse(suiteID)
	if err != nil {
		return nil, errors.New("geçersiz test paketi ID formatı")
	}

	suite, err := s.queries.GetReplayTestSuiteByID(ctx, pgtype.UUID{Bytes: sUUID, Valid: true})
	if err != nil {
		return nil, errors.New("test paketi bulunamadı")
	}

	var reqIDs []string
	if err := json.Unmarshal(suite.RequestIds, &reqIDs); err != nil || len(reqIDs) == 0 {
		return nil, errors.New("test paketinde yürütülecek istek bulunamadı")
	}

	customHeaders := s.resolveCustomHeaders(suite.CustomHeaders)

	totalSteps := len(reqIDs)

	// Create initial run record
	initialRun, err := s.queries.CreateReplayTestRun(ctx, database.CreateReplayTestRunParams{
		SuiteID:        suite.ID,
		Status:         "RUNNING",
		TotalSteps:     int32(totalSteps),
		PassedSteps:    0,
		FailedSteps:    0,
		TotalLatencyMs: 0,
		StepResults:    []byte("[]"),
	})
	if err != nil {
		return nil, fmt.Errorf("test koşusu başlatılamadı: %w", err)
	}

	runIDStr := uuid.UUID(initialRun.ID.Bytes).String()
	stepResults := make([]TestSuiteStepResult, 0, totalSteps)
	passedSteps := 0
	failedSteps := 0
	totalLatencyMs := 0

	for idx, reqID := range reqIDs {
		targetURL := suite.TargetUrl.String
		if targetURL == "" {
			reqUUID, parseErr := uuid.Parse(reqID)
			if parseErr == nil {
				if reqRec, reqErr := s.queries.GetCapturedRequestByID(ctx, pgtype.UUID{Bytes: reqUUID, Valid: true}); reqErr == nil {
					if ep, epErr := s.queries.GetEndpointByIDOnly(ctx, reqRec.EndpointID); epErr == nil && ep.UpstreamUrl.Valid && ep.UpstreamUrl.String != "" {
						targetURL = ep.UpstreamUrl.String
					}
				}
			}
		}

		stepRes := TestSuiteStepResult{
			StepIndex: idx + 1,
			RequestID: reqID,
			TargetURL: targetURL,
		}

		if targetURL == "" {
			stepRes.Status = "FAILED"
			stepRes.ErrorMessage = "Hedef URL belirtilmemiş ve istek için endpoint upstream URL bulunamadı"
			failedSteps++
			stepResults = append(stepResults, stepRes)
			continue
		}

		replayRes, replayErr := s.replayService.ExecuteReplay(ctx, ExecuteReplayParams{
			SourceRequestId:     reqID,
			TargetURL:           targetURL,
			Environment:         suite.TargetEnvironment,
			CustomHeaders:       customHeaders,
			Justification:       fmt.Sprintf("Test Paketi [%s] Koşusu Adım #%d", suite.Name, idx+1),
			OverrideIdempotency: true,
			RenewIdempotency:    suite.RenewIdempotency,
			UserID:              userID,
			ClientIP:            clientIP,
		})

		if replayErr != nil || replayRes.Status == "FAILED" {
			stepRes.Status = "FAILED"
			if replayErr != nil {
				stepRes.ErrorMessage = replayErr.Error()
			} else {
				stepRes.ErrorMessage = replayRes.ResponseBody
			}
			failedSteps++
		} else {
			stepRes.ResponseStatus = replayRes.ResponseStatus
			stepRes.LatencyMs = replayRes.LatencyMs
			stepRes.Replacements = replayRes.Replacements
			totalLatencyMs += int(replayRes.LatencyMs)

			if replayRes.ResponseStatus >= 200 && replayRes.ResponseStatus < 500 {
				stepRes.Status = "PASSED"
				passedSteps++
			} else {
				stepRes.Status = "FAILED"
				stepRes.ErrorMessage = fmt.Sprintf("Upstream HTTP %d hatası döndü", replayRes.ResponseStatus)
				failedSteps++
			}
		}

		stepResults = append(stepResults, stepRes)
	}

	finalStatus := "PASSED"
	if failedSteps > 0 && passedSteps == 0 {
		finalStatus = "FAILED"
	} else if failedSteps > 0 && passedSteps > 0 {
		finalStatus = "PARTIAL"
	}

	stepResultsJSON, _ := json.Marshal(stepResults)
	now := time.Now()

	updatedRun, err := s.queries.UpdateReplayTestRunResult(ctx, database.UpdateReplayTestRunResultParams{
		ID:             initialRun.ID,
		Status:         finalStatus,
		PassedSteps:    int32(passedSteps),
		FailedSteps:    int32(failedSteps),
		TotalLatencyMs: int32(totalLatencyMs),
		StepResults:    stepResultsJSON,
		CompletedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to update test run result")
	}

	return &TestSuiteRunReport{
		RunID:          runIDStr,
		SuiteID:        uuid.UUID(suite.ID.Bytes).String(),
		SuiteName:      suite.Name,
		Status:         finalStatus,
		TotalSteps:     totalSteps,
		PassedSteps:    passedSteps,
		FailedSteps:    failedSteps,
		TotalLatencyMs: totalLatencyMs,
		StepResults:    stepResults,
		CreatedAt:      updatedRun.CreatedAt.Time.Format(time.RFC3339),
		CompletedAt:    now.Format(time.RFC3339),
	}, nil
}
