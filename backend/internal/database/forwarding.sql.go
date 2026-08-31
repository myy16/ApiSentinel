package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const claimPendingOutboxJobs = `-- name: ClaimPendingOutboxJobs :many
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
RETURNING id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at
`

type ClaimPendingOutboxJobsParams struct {
	LockedBy pgtype.Text `json:"locked_by"`
	Limit    int32       `json:"limit"`
}

func (q *Queries) ClaimPendingOutboxJobs(ctx context.Context, arg ClaimPendingOutboxJobsParams) ([]ForwardingDlq, error) {
	rows, err := q.db.Query(ctx, claimPendingOutboxJobs, arg.LockedBy, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ForwardingDlq
	for rows.Next() {
		var i ForwardingDlq
		if err := rows.Scan(
			&i.ID,
			&i.EndpointID,
			&i.RequestID,
			&i.TargetUrl,
			&i.Attempts,
			&i.MaxRetries,
			&i.LastError,
			&i.Payload,
			&i.Status,
			&i.LockedAt,
			&i.LockedBy,
			&i.NextRetryAt,
			&i.CreatedAt,
			&i.LastAttemptAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const completeOutboxJob = `-- name: CompleteOutboxJob :one
UPDATE forwarding_dlq
SET status = 'SENT',
    locked_at = NULL,
    locked_by = NULL,
    last_attempt_at = NOW()
WHERE id = $1
RETURNING id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at
`

func (q *Queries) CompleteOutboxJob(ctx context.Context, id pgtype.UUID) (ForwardingDlq, error) {
	row := q.db.QueryRow(ctx, completeOutboxJob, id)
	var i ForwardingDlq
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.RequestID,
		&i.TargetUrl,
		&i.Attempts,
		&i.MaxRetries,
		&i.LastError,
		&i.Payload,
		&i.Status,
		&i.LockedAt,
		&i.LockedBy,
		&i.NextRetryAt,
		&i.CreatedAt,
		&i.LastAttemptAt,
	)
	return i, err
}

const createDLQRecord = `-- name: CreateDLQRecord :one
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
RETURNING id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at
`

type CreateDLQRecordParams struct {
	EndpointID pgtype.UUID `json:"endpoint_id"`
	RequestID  pgtype.UUID `json:"request_id"`
	TargetUrl  string      `json:"target_url"`
	Attempts   int32       `json:"attempts"`
	LastError  pgtype.Text `json:"last_error"`
	Payload    pgtype.Text `json:"payload"`
	Status     string      `json:"status"`
}

func (q *Queries) CreateDLQRecord(ctx context.Context, arg CreateDLQRecordParams) (ForwardingDlq, error) {
	row := q.db.QueryRow(ctx, createDLQRecord,
		arg.EndpointID,
		arg.RequestID,
		arg.TargetUrl,
		arg.Attempts,
		arg.LastError,
		arg.Payload,
		arg.Status,
	)
	var i ForwardingDlq
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.RequestID,
		&i.TargetUrl,
		&i.Attempts,
		&i.MaxRetries,
		&i.LastError,
		&i.Payload,
		&i.Status,
		&i.LockedAt,
		&i.LockedBy,
		&i.NextRetryAt,
		&i.CreatedAt,
		&i.LastAttemptAt,
	)
	return i, err
}

const createOutboxJob = `-- name: CreateOutboxJob :one
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
RETURNING id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at
`

type CreateOutboxJobParams struct {
	EndpointID pgtype.UUID `json:"endpoint_id"`
	RequestID  pgtype.UUID `json:"request_id"`
	TargetUrl  string      `json:"target_url"`
	Payload    pgtype.Text `json:"payload"`
	MaxRetries int32       `json:"max_retries"`
}

func (q *Queries) CreateOutboxJob(ctx context.Context, arg CreateOutboxJobParams) (ForwardingDlq, error) {
	row := q.db.QueryRow(ctx, createOutboxJob,
		arg.EndpointID,
		arg.RequestID,
		arg.TargetUrl,
		arg.Payload,
		arg.MaxRetries,
	)
	var i ForwardingDlq
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.RequestID,
		&i.TargetUrl,
		&i.Attempts,
		&i.MaxRetries,
		&i.LastError,
		&i.Payload,
		&i.Status,
		&i.LockedAt,
		&i.LockedBy,
		&i.NextRetryAt,
		&i.CreatedAt,
		&i.LastAttemptAt,
	)
	return i, err
}

const deleteDLQRecordsByEndpoint = `-- name: DeleteDLQRecordsByEndpoint :exec
DELETE FROM forwarding_dlq
WHERE endpoint_id = $1
`

func (q *Queries) DeleteDLQRecordsByEndpoint(ctx context.Context, endpointID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteDLQRecordsByEndpoint, endpointID)
	return err
}

const failOutboxJob = `-- name: FailOutboxJob :one
UPDATE forwarding_dlq
SET status = $2,
    attempts = attempts + 1,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $3,
    next_retry_at = $4,
    last_attempt_at = NOW()
WHERE id = $1
RETURNING id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at
`

type FailOutboxJobParams struct {
	ID          pgtype.UUID        `json:"id"`
	Status      string             `json:"status"`
	LastError   pgtype.Text        `json:"last_error"`
	NextRetryAt pgtype.Timestamptz `json:"next_retry_at"`
}

func (q *Queries) FailOutboxJob(ctx context.Context, arg FailOutboxJobParams) (ForwardingDlq, error) {
	row := q.db.QueryRow(ctx, failOutboxJob,
		arg.ID,
		arg.Status,
		arg.LastError,
		arg.NextRetryAt,
	)
	var i ForwardingDlq
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.RequestID,
		&i.TargetUrl,
		&i.Attempts,
		&i.MaxRetries,
		&i.LastError,
		&i.Payload,
		&i.Status,
		&i.LockedAt,
		&i.LockedBy,
		&i.NextRetryAt,
		&i.CreatedAt,
		&i.LastAttemptAt,
	)
	return i, err
}

const getDLQRecordByID = `-- name: GetDLQRecordByID :one
SELECT id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at FROM forwarding_dlq
WHERE id = $1
`

func (q *Queries) GetDLQRecordByID(ctx context.Context, id pgtype.UUID) (ForwardingDlq, error) {
	row := q.db.QueryRow(ctx, getDLQRecordByID, id)
	var i ForwardingDlq
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.RequestID,
		&i.TargetUrl,
		&i.Attempts,
		&i.MaxRetries,
		&i.LastError,
		&i.Payload,
		&i.Status,
		&i.LockedAt,
		&i.LockedBy,
		&i.NextRetryAt,
		&i.CreatedAt,
		&i.LastAttemptAt,
	)
	return i, err
}

const getForwardingConfigByEndpoint = `-- name: GetForwardingConfigByEndpoint :one
SELECT id, endpoint_id, target_url, max_retries, timeout_ms, custom_headers, is_enabled, created_at FROM forwarding_configs
WHERE endpoint_id = $1
`

func (q *Queries) GetForwardingConfigByEndpoint(ctx context.Context, endpointID pgtype.UUID) (ForwardingConfig, error) {
	row := q.db.QueryRow(ctx, getForwardingConfigByEndpoint, endpointID)
	var i ForwardingConfig
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.TargetUrl,
		&i.MaxRetries,
		&i.TimeoutMs,
		&i.CustomHeaders,
		&i.IsEnabled,
		&i.CreatedAt,
	)
	return i, err
}

const listDLQRecordsByEndpoint = `-- name: ListDLQRecordsByEndpoint :many
SELECT id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at FROM forwarding_dlq
WHERE endpoint_id = $1
ORDER BY created_at DESC
LIMIT 100
`

func (q *Queries) ListDLQRecordsByEndpoint(ctx context.Context, endpointID pgtype.UUID) ([]ForwardingDlq, error) {
	rows, err := q.db.Query(ctx, listDLQRecordsByEndpoint, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ForwardingDlq
	for rows.Next() {
		var i ForwardingDlq
		if err := rows.Scan(
			&i.ID,
			&i.EndpointID,
			&i.RequestID,
			&i.TargetUrl,
			&i.Attempts,
			&i.MaxRetries,
			&i.LastError,
			&i.Payload,
			&i.Status,
			&i.LockedAt,
			&i.LockedBy,
			&i.NextRetryAt,
			&i.CreatedAt,
			&i.LastAttemptAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const recoverStaleOutboxJobs = `-- name: RecoverStaleOutboxJobs :exec
UPDATE forwarding_dlq
SET status = 'RETRY_WAIT',
    locked_at = NULL,
    locked_by = NULL,
    next_retry_at = NOW()
WHERE status = 'PROCESSING'
  AND locked_at < NOW() - INTERVAL '2 minutes'
`

func (q *Queries) RecoverStaleOutboxJobs(ctx context.Context) error {
	_, err := q.db.Exec(ctx, recoverStaleOutboxJobs)
	return err
}

const updateDLQStatus = `-- name: UpdateDLQStatus :one
UPDATE forwarding_dlq
SET status = $2,
    attempts = attempts + 1,
    last_attempt_at = NOW()
WHERE id = $1
RETURNING id, endpoint_id, request_id, target_url, attempts, max_retries, last_error, payload, status, locked_at, locked_by, next_retry_at, created_at, last_attempt_at
`

type UpdateDLQStatusParams struct {
	ID     pgtype.UUID `json:"id"`
	Status string      `json:"status"`
}

func (q *Queries) UpdateDLQStatus(ctx context.Context, arg UpdateDLQStatusParams) (ForwardingDlq, error) {
	row := q.db.QueryRow(ctx, updateDLQStatus, arg.ID, arg.Status)
	var i ForwardingDlq
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.RequestID,
		&i.TargetUrl,
		&i.Attempts,
		&i.MaxRetries,
		&i.LastError,
		&i.Payload,
		&i.Status,
		&i.LockedAt,
		&i.LockedBy,
		&i.NextRetryAt,
		&i.CreatedAt,
		&i.LastAttemptAt,
	)
	return i, err
}

const upsertForwardingConfig = `-- name: UpsertForwardingConfig :one
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
RETURNING id, endpoint_id, target_url, max_retries, timeout_ms, custom_headers, is_enabled, created_at
`

type UpsertForwardingConfigParams struct {
	EndpointID    pgtype.UUID `json:"endpoint_id"`
	TargetUrl     string      `json:"target_url"`
	MaxRetries    int32       `json:"max_retries"`
	TimeoutMs     int32       `json:"timeout_ms"`
	CustomHeaders []byte      `json:"custom_headers"`
	IsEnabled     bool        `json:"is_enabled"`
}

func (q *Queries) UpsertForwardingConfig(ctx context.Context, arg UpsertForwardingConfigParams) (ForwardingConfig, error) {
	row := q.db.QueryRow(ctx, upsertForwardingConfig,
		arg.EndpointID,
		arg.TargetUrl,
		arg.MaxRetries,
		arg.TimeoutMs,
		arg.CustomHeaders,
		arg.IsEnabled,
	)
	var i ForwardingConfig
	err := row.Scan(
		&i.ID,
		&i.EndpointID,
		&i.TargetUrl,
		&i.MaxRetries,
		&i.TimeoutMs,
		&i.CustomHeaders,
		&i.IsEnabled,
		&i.CreatedAt,
	)
	return i, err
}

const verifyDLQRecordOwnership = `-- name: VerifyDLQRecordOwnership :one
SELECT d.id
FROM forwarding_dlq d
JOIN endpoints e ON e.id = d.endpoint_id
JOIN projects p ON p.id = e.project_id
WHERE d.id = $1 AND p.organization_id = $2
LIMIT 1
`

type VerifyDLQRecordOwnershipParams struct {
	ID             pgtype.UUID `json:"id"`
	OrganizationID pgtype.UUID `json:"organization_id"`
}

func (q *Queries) VerifyDLQRecordOwnership(ctx context.Context, arg VerifyDLQRecordOwnershipParams) (pgtype.UUID, error) {
	row := q.db.QueryRow(ctx, verifyDLQRecordOwnership, arg.ID, arg.OrganizationID)
	var id pgtype.UUID
	err := row.Scan(&id)
	return id, err
}
