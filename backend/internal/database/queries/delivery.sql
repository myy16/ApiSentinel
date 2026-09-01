-- name: CreateDeliveryJob :one
INSERT INTO delivery_jobs (
    endpoint_id,
    request_id,
    target_url,
    status,
    max_retries,
    idempotency_key,
    payload_mode,
    next_retry_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, NOW()
)
RETURNING *;

-- name: ClaimPendingDeliveryJobs :many
UPDATE delivery_jobs
SET status = 'PROCESSING',
    locked_at = NOW(),
    locked_by = $1,
    updated_at = NOW()
WHERE id IN (
    SELECT id FROM delivery_jobs
    WHERE status IN ('PENDING', 'RETRY_WAIT')
      AND next_retry_at <= NOW()
      AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '2 minutes')
    ORDER BY next_retry_at ASC
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: RecordDeliveryAttempt :one
INSERT INTO delivery_attempts (
    job_id,
    attempt_number,
    started_at,
    finished_at,
    latency_ms,
    response_status_code,
    error_type,
    error_message,
    request_headers_sent,
    response_headers_received,
    response_body_snippet
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: CompleteDeliveryJob :one
UPDATE delivery_jobs
SET status = 'DELIVERED',
    attempts = attempts + 1,
    locked_at = NULL,
    locked_by = NULL,
    last_error = NULL,
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailDeliveryJob :one
UPDATE delivery_jobs
SET status = $2,
    attempts = attempts + 1,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $3,
    next_retry_at = $4,
    updated_at = NOW(),
    completed_at = CASE WHEN $2 = 'DEAD_LETTER' THEN NOW() ELSE completed_at END
WHERE id = $1
RETURNING *;

-- name: RecoverStaleDeliveryJobs :exec
UPDATE delivery_jobs
SET status = 'RETRY_WAIT',
    locked_at = NULL,
    locked_by = NULL,
    next_retry_at = NOW(),
    updated_at = NOW()
WHERE status = 'PROCESSING'
  AND locked_at < NOW() - INTERVAL '2 minutes';

-- name: GetDeliveryJobByID :one
SELECT * FROM delivery_jobs
WHERE id = $1;

-- name: GetDeliveryJobByRequestID :one
SELECT * FROM delivery_jobs
WHERE request_id = $1
LIMIT 1;

-- name: ListDeliveryJobsByEndpoint :many
SELECT * FROM delivery_jobs
WHERE endpoint_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDeliveryAttemptsByJobID :many
SELECT * FROM delivery_attempts
WHERE job_id = $1
ORDER BY attempt_number ASC;

-- name: VerifyDeliveryJobOwnership :one
SELECT j.id
FROM delivery_jobs j
JOIN endpoints e ON e.id = j.endpoint_id
JOIN projects p ON p.id = e.project_id
WHERE j.id = $1 AND p.organization_id = $2
LIMIT 1;

-- name: RequeueDeliveryJob :one
UPDATE delivery_jobs
SET status = 'PENDING',
    attempts = 0,
    locked_at = NULL,
    locked_by = NULL,
    next_retry_at = NOW(),
    completed_at = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
