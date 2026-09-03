-- name: CreateReplayJob :one
INSERT INTO replay_jobs (source_request_id, target_type, target_url, environment, custom_headers, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, source_request_id, target_type, target_url, environment, custom_headers, status, response_status, response_body, latency_ms, created_at, completed_at;

-- name: UpdateReplayJobResult :one
UPDATE replay_jobs
SET status = $2,
    response_status = $3,
    response_body = $4,
    latency_ms = $5,
    completed_at = $6
WHERE id = $1
RETURNING id, source_request_id, target_type, target_url, environment, custom_headers, status, response_status, response_body, latency_ms, created_at, completed_at;

-- name: ListReplayJobsByProject :many
SELECT rj.id, rj.source_request_id, rj.target_type, rj.target_url, rj.environment, rj.custom_headers, rj.status,
       rj.response_status, rj.response_body, rj.latency_ms, rj.created_at, rj.completed_at,
       cr.http_method, cr.request_id, cr.response_status as original_response_status, e.name as endpoint_name
FROM replay_jobs rj
JOIN captured_requests cr ON rj.source_request_id = cr.id
JOIN endpoints e ON cr.endpoint_id = e.id
WHERE e.project_id = $1
ORDER BY rj.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetReplayJobByID :one
SELECT rj.id, rj.source_request_id, rj.target_type, rj.target_url, rj.environment, rj.custom_headers, rj.status,
       rj.response_status, rj.response_body, rj.latency_ms, rj.created_at, rj.completed_at,
       cr.http_method, cr.request_id, cr.response_status as original_response_status, cr.headers as original_headers,
       cr.parsed_json as original_payload, e.name as endpoint_name, e.slug as endpoint_slug
FROM replay_jobs rj
JOIN captured_requests cr ON rj.source_request_id = cr.id
JOIN endpoints e ON cr.endpoint_id = e.id
WHERE rj.id = $1;
