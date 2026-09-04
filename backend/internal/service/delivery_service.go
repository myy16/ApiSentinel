package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/delivery"
	"github.com/apisentinel/apisentinel/internal/forwarding"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/apisentinel/apisentinel/internal/security/redaction"
	"github.com/apisentinel/apisentinel/internal/security/ssrf"
	"github.com/apisentinel/apisentinel/internal/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

// DeliveryService manages atomic webhook ingestion, transactional outbox forwarding,
// lease-locked worker execution, attempt telemetry, and crash recovery.
type DeliveryService struct {
	queries        *database.Queries
	httpClient     *http.Client
	workerPool     *worker.Pool
	workerID       string
	encryptionKey  string
	endpointSem    map[string]chan struct{}
	endpointSemMu  sync.Mutex
	maxPerEndpoint int
	alertService   *AlertService
	stopPoller     chan struct{}
	triggerPoller  chan struct{}
}

func (s *DeliveryService) SetAlertService(alertService *AlertService) {
	s.alertService = alertService
}

func NewDeliveryService(
	queries *database.Queries,
	workerPool *worker.Pool,
	encryptionKey string,
) *DeliveryService {
	if workerPool == nil {
		workerPool = worker.NewPool(20, 5000)
	}
	workerID := fmt.Sprintf("delivery-worker-%s", uuid.New().String()[:8])

	svc := &DeliveryService{
		queries:        queries,
		httpClient:     ssrf.NewSafeHTTPClient(10 * time.Second),
		workerPool:     workerPool,
		workerID:       workerID,
		encryptionKey:  encryptionKey,
		endpointSem:    make(map[string]chan struct{}),
		maxPerEndpoint: 5, // Fair scheduling: max 5 concurrent outbound requests per endpoint
		triggerPoller:  make(chan struct{}, 100),
	}

	// Startup Crash Recovery
	if queries != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := queries.RecoverStaleDeliveryJobs(ctx); err != nil {
				log.Warn().Err(err).Msg("Failed to recover stale delivery jobs on startup")
			} else {
				log.Info().Msg("Delivery crash recovery executed: unhandled processing jobs returned to RETRY_WAIT")
			}
		}()
	}

	return svc
}

// IngestAtomicParams defines the payload needed to persist a webhook and its outbox job atomically.
type IngestAtomicParams struct {
	EndpointID     pgtype.UUID
	RequestID      string
	HTTPMethod     string
	HeadersBytes   []byte
	QueryParams    []byte
	MaskedBody     pgtype.Text
	ParsedJSON     []byte
	ClientIP       string
	ResponseStatus int32
	RequestState   delivery.RequestState
	TargetURL      string
	IdempotencyKey string
	PayloadMode    string
	MaxRetries     int
	RawBody        []byte
}

// CreateJobRecord creates a delivery job inside the database.
func (s *DeliveryService) CreateJobRecord(ctx context.Context, params IngestAtomicParams, capturedID pgtype.UUID) (*database.DeliveryJob, error) {
	if params.TargetURL == "" {
		return nil, nil // Not configured for forwarding
	}

	maxRetries := params.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	payloadMode := params.PayloadMode
	if payloadMode != "RAW" {
		payloadMode = "REDACTED"
	}

	var idempKey *string
	if params.IdempotencyKey != "" {
		idempKey = &params.IdempotencyKey
	}

	var pgIdemp pgtype.Text
	if idempKey != nil {
		pgIdemp = pgtype.Text{String: *idempKey, Valid: true}
	}

	job, err := s.queries.CreateDeliveryJob(ctx, database.CreateDeliveryJobParams{
		EndpointID:     params.EndpointID,
		RequestID:      capturedID,
		TargetUrl:      params.TargetURL,
		Status:         string(delivery.DeliveryStatePending),
		MaxRetries:     int32(maxRetries),
		IdempotencyKey: pgIdemp,
		PayloadMode:    payloadMode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create delivery job: %w", err)
	}

	return &job, nil
}

// ProcessJobAsync dispatches a delivery job to the worker pool.
func (s *DeliveryService) ProcessJobAsync(job database.DeliveryJob, method string, headers map[string]string, body []byte) {
	task := func(taskCtx context.Context) {
		bgCtx := taskCtx
		if bgCtx == nil {
			bgCtx = context.Background()
		}
		s.ExecuteJob(bgCtx, job, method, headers, body)
	}

	if s.workerPool != nil {
		if err := s.workerPool.Submit(task); err != nil {
			log.Warn().Err(err).Msg("Worker pool saturated; job remains PENDING for queue worker pickup")
		}
	}
}

// acquireEndpointSemaphore enforces fair scheduling across endpoints so one slow upstream doesn't block others.
func (s *DeliveryService) acquireEndpointSemaphore(endpointID string) func() {
	s.endpointSemMu.Lock()
	ch, exists := s.endpointSem[endpointID]
	if !exists {
		ch = make(chan struct{}, s.maxPerEndpoint)
		s.endpointSem[endpointID] = ch
	}
	s.endpointSemMu.Unlock()

	ch <- struct{}{}
	return func() {
		<-ch
	}
}

// ExecuteJob executes a single forward attempt, evaluates the HTTP response via delivery.EvaluateResponse,
// records attempt telemetry in delivery_attempts, and updates delivery_jobs state.
func (s *DeliveryService) ExecuteJob(ctx context.Context, job database.DeliveryJob, method string, headers map[string]string, bodyBytes []byte) (*database.DeliveryAttempt, error) {
	endpointIDStr := uuid.UUID(job.EndpointID.Bytes).String()
	releaseSem := s.acquireEndpointSemaphore(endpointIDStr)
	defer releaseSem()

	// 1. Validate Target URL via SSRF Guard
	_, ssrfErr := ssrf.ValidateURL(job.TargetUrl)
	if ssrfErr != nil {
		attemptNum := job.Attempts + 1
		reqHeadersJSON, _ := json.Marshal(delivery.RedactHeaderMap(headers))

		attempt, _ := s.queries.RecordDeliveryAttempt(ctx, database.RecordDeliveryAttemptParams{
			JobID:                   job.ID,
			AttemptNumber:           attemptNum,
			StartedAt:               pgtype.Timestamptz{Time: time.Now(), Valid: true},
			FinishedAt:              pgtype.Timestamptz{Time: time.Now(), Valid: true},
			LatencyMs:               0,
			ResponseStatusCode:      pgtype.Int4{Valid: false},
			ErrorType:               pgtype.Text{String: "SSRF_BLOCKED", Valid: true},
			ErrorMessage:            pgtype.Text{String: ssrfErr.Error(), Valid: true},
			RequestHeadersSent:      reqHeadersJSON,
			ResponseHeadersReceived: []byte("{}"),
			ResponseBodySnippet:     pgtype.Text{Valid: false},
		})

		_, _ = s.queries.FailDeliveryJob(ctx, database.FailDeliveryJobParams{
			ID:          job.ID,
			Status:      string(delivery.DeliveryStateDeadLetter),
			LastError:   pgtype.Text{String: "SSRF Guard blocked upstream URL: " + ssrfErr.Error(), Valid: true},
			NextRetryAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
		})

		return &attempt, ssrfErr
	}

	// 2. Prepare Outbound Request with Dynamic Timeout & Decrypted Custom Headers
	timeout := 10 * time.Second
	var customHeaders map[string]string
	if cfg, cfgErr := s.queries.GetForwardingConfigByEndpoint(ctx, job.EndpointID); cfgErr == nil {
		if cfg.TimeoutMs > 0 {
			timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
		}
		customHeaders = s.resolveCustomHeaders(cfg)
	}

	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Safe Header Sanitization for request
	safeHeaders := forwarding.FilterHeaders(headers)
	for k, v := range customHeaders {
		safeHeaders[k] = v
	}

	cleanPayload := bodyBytes
	if job.PayloadMode != "RAW" {
		masked, _ := redaction.Payload(bodyBytes)
		cleanPayload = []byte(masked)
	}

	req, err := http.NewRequestWithContext(attemptCtx, method, job.TargetUrl, bytes.NewReader(cleanPayload))
	if err != nil {
		return nil, err
	}

	for k, v := range safeHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-ApiSentinel-Forwarded", "true")
	req.Header.Set("X-ApiSentinel-Job-ID", uuid.UUID(job.ID.Bytes).String())

	startTime := time.Now()
	resp, doErr := s.httpClient.Do(req)
	latency := time.Since(startTime).Milliseconds()
	finishedTime := time.Now()

	var statusCode int
	var respHeaders http.Header
	var respBodySnippet string

	if resp != nil {
		statusCode = resp.StatusCode
		respHeaders = resp.Header

		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		respBodySnippet = delivery.SanitizeBodySnippet(respBytes, delivery.DefaultMaxBodySnippetBytes)
	}

	// 3. Evaluate Decision via Pure Delivery Policy Engine
	opts := delivery.DefaultRetryOptions()
	opts.MaxRetries = int(job.MaxRetries)

	eval := delivery.EvaluateResponse(statusCode, doErr, int(job.Attempts+1), respHeaders, opts)

	// 4. Record Attempt Telemetry in delivery_attempts
	reqHeadersSanitized, _ := json.Marshal(delivery.RedactHeaderMap(safeHeaders))
	respHeadersSanitized, _ := json.Marshal(delivery.RedactHeaders(respHeaders))

	var errTypeStr, errMsgStr string
	if doErr != nil {
		errTypeStr = "NETWORK_ERROR"
		errMsgStr = doErr.Error()
	} else if statusCode >= 400 {
		errTypeStr = "HTTP_ERROR"
		errMsgStr = fmt.Sprintf("Upstream returned HTTP %d", statusCode)
	}

	var pgRespStatus pgtype.Int4
	if statusCode > 0 {
		pgRespStatus = pgtype.Int4{Int32: int32(statusCode), Valid: true}
	}

	var pgErrType pgtype.Text
	if errTypeStr != "" {
		pgErrType = pgtype.Text{String: errTypeStr, Valid: true}
	}

	var pgErrMsg pgtype.Text
	if errMsgStr != "" {
		pgErrMsg = pgtype.Text{String: errMsgStr, Valid: true}
	}

	var pgRespSnippet pgtype.Text
	if respBodySnippet != "" {
		pgRespSnippet = pgtype.Text{String: respBodySnippet, Valid: true}
	}

	attempt, err := s.queries.RecordDeliveryAttempt(ctx, database.RecordDeliveryAttemptParams{
		JobID:                   job.ID,
		AttemptNumber:           job.Attempts + 1,
		StartedAt:               pgtype.Timestamptz{Time: startTime, Valid: true},
		FinishedAt:              pgtype.Timestamptz{Time: finishedTime, Valid: true},
		LatencyMs:               int32(latency),
		ResponseStatusCode:      pgRespStatus,
		ErrorType:               pgErrType,
		ErrorMessage:            pgErrMsg,
		RequestHeadersSent:      reqHeadersSanitized,
		ResponseHeadersReceived: respHeadersSanitized,
		ResponseBodySnippet:     pgRespSnippet,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to record delivery attempt telemetry")
	}

	// 5. Update Job State
	if eval.NextState == delivery.DeliveryStateDelivered {
		_, _ = s.queries.CompleteDeliveryJob(ctx, job.ID)
		_ = s.queries.UpdateRequestProcessingStatus(ctx, database.UpdateRequestProcessingStatusParams{
			ID:               job.RequestID,
			ProcessingStatus: "DELIVERED",
		})
	} else {
		nextRetry := time.Now().Add(eval.BackoffDelay)
		if eval.NextState == delivery.DeliveryStateDeadLetter {
			nextRetry = time.Now().Add(24 * time.Hour)
		}

		_, _ = s.queries.FailDeliveryJob(ctx, database.FailDeliveryJobParams{
			ID:          job.ID,
			Status:      string(eval.NextState),
			LastError:   pgtype.Text{String: eval.ReasonSummary, Valid: true},
			NextRetryAt: pgtype.Timestamptz{Time: nextRetry, Valid: true},
		})

		procStatus := "RETRY_WAIT"
		if eval.NextState == delivery.DeliveryStateDeadLetter {
			procStatus = "DEAD_LETTER"
		}
		_ = s.queries.UpdateRequestProcessingStatus(ctx, database.UpdateRequestProcessingStatusParams{
			ID:               job.RequestID,
			ProcessingStatus: procStatus,
		})

		// Trigger delivery anomaly / digest alerting
		if s.alertService != nil {
			endpoint, _ := s.queries.GetEndpointByIDOnly(ctx, job.EndpointID)
			epName := endpoint.Name
			if epName == "" {
				epName = endpoint.Slug
			}
			projID := uuid.UUID(endpoint.ProjectID.Bytes).String()
			epID := uuid.UUID(job.EndpointID.Bytes).String()
			isDLQ := (eval.NextState == delivery.DeliveryStateDeadLetter)
			s.alertService.RecordDeliveryFailure(projID, epName, epID, epName, job.TargetUrl, statusCode, eval.ReasonSummary, isDLQ)
		}
	}

	return &attempt, doErr
}

// PollAndProcessQueue runs a queue batch claim for outbox worker polling.
func (s *DeliveryService) PollAndProcessQueue(ctx context.Context, batchSize int32) (int, error) {
	if batchSize <= 0 {
		batchSize = 10
	}

	jobs, err := s.queries.ClaimPendingDeliveryJobs(ctx, database.ClaimPendingDeliveryJobsParams{
		LockedBy: pgtype.Text{String: s.workerID, Valid: true},
		Limit:    batchSize,
	})
	if err != nil {
		return 0, err
	}

	for _, j := range jobs {
		jobCopy := j
		// Fetch request details
		req, reqErr := s.queries.GetCapturedRequestByID(ctx, jobCopy.RequestID)
		if reqErr != nil {
			continue
		}

		var headers map[string]string
		_ = json.Unmarshal(req.Headers, &headers)

		body := []byte(req.MaskedBody.String)
		s.ProcessJobAsync(jobCopy, req.HttpMethod, headers, body)
	}

	return len(jobs), nil
}

// TriggerQueue sends an immediate non-blocking notification to the queue poller
// to claim and process pending delivery jobs without waiting for the next ticker tick.
func (s *DeliveryService) TriggerQueue() {
	if s.triggerPoller != nil {
		select {
		case s.triggerPoller <- struct{}{}:
		default:
			// Poller is already triggered or processing; skip
		}
	}
}

// StartQueuePoller starts a background goroutine that periodically polls for
// PENDING/RETRY_WAIT delivery jobs whose next_retry_at has passed.
// All delivery jobs are claimed atomically via FOR UPDATE SKIP LOCKED to guarantee
// no double delivery race conditions.
func (s *DeliveryService) StartQueuePoller(ctx context.Context, interval time.Duration, batchSize int32) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 10
	}

	s.stopPoller = make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		log.Info().
			Dur("interval", interval).
			Int32("batchSize", batchSize).
			Msg("Delivery queue poller started")

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Delivery queue poller stopped (context cancelled)")
				return
			case <-s.stopPoller:
				log.Info().Msg("Delivery queue poller stopped (explicit stop)")
				return
			case <-s.triggerPoller:
				for {
					processed, err := s.PollAndProcessQueue(ctx, batchSize)
					if err != nil {
						log.Warn().Err(err).Msg("Delivery queue triggered poll failed")
						break
					}
					if processed < int(batchSize) {
						break
					}
				}
			case <-ticker.C:
				for {
					processed, err := s.PollAndProcessQueue(ctx, batchSize)
					if err != nil {
						log.Warn().Err(err).Msg("Delivery queue periodic poll cycle failed")
						break
					}
					if processed == 0 || processed < int(batchSize) {
						break
					}
				}
			}
		}
	}()
}

// StopQueuePoller signals the background queue poller goroutine to stop.
func (s *DeliveryService) StopQueuePoller() {
	if s.stopPoller != nil {
		close(s.stopPoller)
	}
}

// resolveCustomHeaders decrypts stored headers if encrypted, or unmarshals JSON directly
func (s *DeliveryService) resolveCustomHeaders(cfg database.ForwardingConfig) map[string]string {
	if len(cfg.CustomHeaders) == 0 {
		return nil
	}

	// 1. Check for encrypted JSON envelope {"_encrypted": "..."}
	var env map[string]string
	if err := json.Unmarshal(cfg.CustomHeaders, &env); err == nil {
		if encVal, ok := env["_encrypted"]; ok && encVal != "" {
			if s.encryptionKey != "" {
				decrypted, dErr := envelope.Decrypt(s.encryptionKey, encVal)
				if dErr == nil && decrypted != "" {
					var headers map[string]string
					if err := json.Unmarshal([]byte(decrypted), &headers); err == nil {
						return headers
					}
				}
			}
		} else {
			// Direct unencrypted map
			return env
		}
	}

	// 2. Direct string decrypt fallback
	if s.encryptionKey != "" {
		decrypted, err := envelope.Decrypt(s.encryptionKey, string(cfg.CustomHeaders))
		if err == nil && decrypted != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(decrypted), &headers); err == nil {
				return headers
			}
		}
	}

	var headers map[string]string
	_ = json.Unmarshal(cfg.CustomHeaders, &headers)
	return headers
}
