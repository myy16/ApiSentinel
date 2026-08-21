-- name: CreateReplayJob :one
INSERT INTO replay_jobs (source_request_id, target_type, target_url, status)
VALUES ($1, $2, $3, $4)
RETURNING id, source_request_id, target_type, target_url, status, response_status, response_body, created_at, completed_at;

-- name: UpdateReplayJobResult :one
UPDATE replay_jobs
SET status = $2,
    response_status = $3,
    response_body = $4,
    completed_at = $5
WHERE id = $1
RETURNING id, source_request_id, target_type, target_url, status, response_status, response_body, created_at, completed_at;

-- name: ListReplayJobsByProject :many
SELECT rj.id, rj.source_request_id, rj.target_type, rj.target_url, rj.status,
       rj.response_status, rj.response_body, rj.created_at, rj.completed_at,
       cr.http_method, cr.request_id, e.name as endpoint_name
FROM replay_jobs rj
JOIN captured_requests cr ON rj.source_request_id = cr.id
JOIN endpoints e ON cr.endpoint_id = e.id
WHERE e.project_id = $1
ORDER BY rj.created_at DESC
LIMIT $2 OFFSET $3;
