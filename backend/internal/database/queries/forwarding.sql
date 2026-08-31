-- name: UpsertForwardingConfig :one
INSERT INTO forwarding_configs (
    endpoint_id,
    target_url,
    max_retries,
    timeout_ms,
    custom_headers,
    is_enabled
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (endpoint_id) DO UPDATE SET
    target_url = EXCLUDED.target_url,
    max_retries = EXCLUDED.max_retries,
    timeout_ms = EXCLUDED.timeout_ms,
    custom_headers = EXCLUDED.custom_headers,
    is_enabled = EXCLUDED.is_enabled
RETURNING *;

-- name: GetForwardingConfigByEndpoint :one
SELECT * FROM forwarding_configs
WHERE endpoint_id = $1;

-- name: CreateDLQRecord :one
INSERT INTO forwarding_dlq (
    endpoint_id,
    request_id,
    target_url,
    attempts,
    last_error,
    payload,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: CreateOutboxJob :one
INSERT INTO forwarding_dlq (
    endpoint_id,
    request_id,
    target_url,
    payload,
    status,
    max_retries,
    next_retry_at
) VALUES (
    $1, $2, $3, $4, 'PENDING', $5, NOW()
)
RETURNING *;

-- name: ClaimPendingOutboxJobs :many
UPDATE forwarding_dlq
SET status = 'PROCESSING',
    locked_at = NOW(),
    locked_by = $1
WHERE id IN (
    SELECT id FROM forwarding_dlq
    WHERE status IN ('PENDING', 'RETRY_WAIT')
      AND next_retry_at <= NOW()
      AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '2 minutes')
    ORDER BY next_retry_at ASC
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: CompleteOutboxJob :one
UPDATE forwarding_dlq
SET status = 'SENT',
    locked_at = NULL,
    locked_by = NULL,
    last_attempt_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailOutboxJob :one
UPDATE forwarding_dlq
SET status = $2,
    attempts = attempts + 1,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $3,
    next_retry_at = $4,
    last_attempt_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RecoverStaleOutboxJobs :exec
UPDATE forwarding_dlq
SET status = 'RETRY_WAIT',
    locked_at = NULL,
    locked_by = NULL,
    next_retry_at = NOW()
WHERE status = 'PROCESSING'
  AND locked_at < NOW() - INTERVAL '2 minutes';

-- name: ListDLQRecordsByEndpoint :many
SELECT * FROM forwarding_dlq
WHERE endpoint_id = $1
ORDER BY created_at DESC
LIMIT 100;

-- name: GetDLQRecordByID :one
SELECT * FROM forwarding_dlq
WHERE id = $1;

-- name: VerifyDLQRecordOwnership :one
SELECT d.id
FROM forwarding_dlq d
JOIN endpoints e ON e.id = d.endpoint_id
JOIN projects p ON p.id = e.project_id
WHERE d.id = $1 AND p.organization_id = $2
LIMIT 1;

-- name: UpdateDLQStatus :one
UPDATE forwarding_dlq
SET status = $2,
    attempts = attempts + 1,
    last_attempt_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDLQRecordsByEndpoint :exec
DELETE FROM forwarding_dlq
WHERE endpoint_id = $1;
