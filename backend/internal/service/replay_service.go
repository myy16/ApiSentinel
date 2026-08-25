package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/ssrf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type ReplayService struct {
	queries    *database.Queries
	httpClient *http.Client
}

func NewReplayService(queries *database.Queries) *ReplayService {
	return &ReplayService{
		queries: queries,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type ReplayResultResponse struct {
	JobID          string `json:"jobId"`
	Status         string `json:"status"`
	ResponseStatus int    `json:"responseStatus"`
	ResponseBody   string `json:"responseBody"`
	LatencyMs      int64  `json:"latencyMs"`
	TargetURL      string `json:"targetUrl"`
	CreatedAt      string `json:"createdAt"`
}

func (s *ReplayService) ExecuteReplay(ctx context.Context, sourceRequestId, targetUrl string) (*ReplayResultResponse, error) {
	// 1. SSRF Validation
	_, err := ssrf.ValidateURL(targetUrl)
	if err != nil {
		return nil, fmt.Errorf("SSRF Guard blocked target URL: %w", err)
	}

	// 2. Fetch original captured request
	reqUUID, err := uuid.Parse(sourceRequestId)
	if err != nil {
		return nil, errors.New("geçersiz istek ID formatı")
	}

	var pgReqId pgtype.UUID
	copy(pgReqId.Bytes[:], reqUUID[:])
	pgReqId.Valid = true

	reqRecord, err := s.queries.GetCapturedRequestByID(ctx, pgReqId)
	if err != nil {
		return nil, errors.New("kayıtlı istek bulunamadı")
	}

	// 3. Create initial Replay Job
	job, err := s.queries.CreateReplayJob(ctx, database.CreateReplayJobParams{
		SourceRequestID: pgReqId,
		TargetType:      "DIRECT_HTTP",
		TargetUrl:       pgtype.Text{String: targetUrl, Valid: true},
		Status:          "RUNNING",
	})
	if err != nil {
		return nil, fmt.Errorf("replay işi oluşturulamadı: %w", err)
	}

	jobIdStr := uuid.UUID(job.ID.Bytes).String()

	// 4. Prepare Outbound HTTP Request
	var bodyReader io.Reader
	if reqRecord.RawBody.Valid && len(reqRecord.RawBody.String) > 0 {
		bodyReader = bytes.NewBufferString(reqRecord.RawBody.String)
	}

	outboundReq, err := http.NewRequestWithContext(ctx, reqRecord.HttpMethod, targetUrl, bodyReader)
	if err != nil {
		return nil, err
	}

	// Restore original headers (DB stores as map[string][]string or map[string]string)
	var headers map[string]interface{}
	if err := json.Unmarshal(reqRecord.Headers, &headers); err == nil {
		for k, v := range headers {
			switch val := v.(type) {
			case string:
				outboundReq.Header.Set(k, val)
			case []interface{}:
				for i, item := range val {
					if strItem, ok := item.(string); ok {
						if i == 0 {
							outboundReq.Header.Set(k, strItem)
						} else {
							outboundReq.Header.Add(k, strItem)
						}
					}
				}
			}
		}
	}
	outboundReq.Header.Set("X-ApiSentinel-Replayed", "true")

	// 5. Execute HTTP Replay
	startTime := time.Now()
	resp, reqErr := s.httpClient.Do(outboundReq)
	latencyMs := time.Since(startTime).Milliseconds()

	var respStatus int = 0
	var respBodyStr string = ""
	status := "COMPLETED"

	if reqErr != nil {
		status = "FAILED"
		respBodyStr = reqErr.Error()
		log.Warn().Err(reqErr).Str("jobId", jobIdStr).Msg("Replay request failed")
	} else {
		respStatus = resp.StatusCode
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		respBodyStr = string(bodyBytes)
	}

	// 6. Update Replay Job with Results
	now := time.Now()
	updatedJob, err := s.queries.UpdateReplayJobResult(ctx, database.UpdateReplayJobResultParams{
		ID:             job.ID,
		Status:         status,
		ResponseStatus: pgtype.Int4{Int32: int32(respStatus), Valid: true},
		ResponseBody:   pgtype.Text{String: respBodyStr, Valid: true},
		CompletedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to update replay job record")
	}

	return &ReplayResultResponse{
		JobID:          jobIdStr,
		Status:         updatedJob.Status,
		ResponseStatus: respStatus,
		ResponseBody:   respBodyStr,
		LatencyMs:      latencyMs,
		TargetURL:      targetUrl,
		CreatedAt:      updatedJob.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *ReplayService) ListReplayJobs(ctx context.Context, projectId string, limit, offset int32) ([]map[string]interface{}, error) {
	projUUID, err := uuid.Parse(projectId)
	if err != nil {
		return nil, err
	}

	var pgProjId pgtype.UUID
	copy(pgProjId.Bytes[:], projUUID[:])
	pgProjId.Valid = true

	jobs, err := s.queries.ListReplayJobsByProject(ctx, database.ListReplayJobsByProjectParams{
		ProjectID: pgProjId,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	var res []map[string]interface{}
	for _, j := range jobs {
		var status int32 = 0
		if j.ResponseStatus.Valid {
			status = j.ResponseStatus.Int32
		}
		var target string = ""
		if j.TargetUrl.Valid {
			target = j.TargetUrl.String
		}
		var respBody string = ""
		if j.ResponseBody.Valid {
			respBody = j.ResponseBody.String
		}

		res = append(res, map[string]interface{}{
			"id":              uuid.UUID(j.ID.Bytes).String(),
			"sourceRequestId": uuid.UUID(j.SourceRequestID.Bytes).String(),
			"requestId":       j.RequestID,
			"httpMethod":      j.HttpMethod,
			"endpointName":    j.EndpointName,
			"targetUrl":       target,
			"status":          j.Status,
			"responseStatus":  status,
			"responseBody":    respBody,
			"createdAt":       j.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, nil
}
