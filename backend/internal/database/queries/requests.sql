-- name: CreateCapturedRequest :one
INSERT INTO captured_requests (
    endpoint_id, request_id, http_method, headers, query_params,
    raw_body, masked_body, parsed_json, client_ip, response_status, processing_status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING id, endpoint_id, request_id, http_method, headers, query_params, raw_body, masked_body, parsed_json, client_ip, response_status, processing_status, created_at;

-- name: ListRequestsByEndpoint :many
SELECT id, endpoint_id, request_id, http_method, headers, query_params,
       raw_body, masked_body, parsed_json, client_ip, response_status, processing_status, created_at
FROM captured_requests
WHERE endpoint_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRequestsByProject :many
SELECT r.id, r.endpoint_id, r.request_id, r.http_method, r.headers, r.query_params,
       r.raw_body, r.masked_body, r.parsed_json, r.client_ip, r.response_status, r.processing_status, r.created_at,
       e.name as endpoint_name, e.slug as endpoint_slug
FROM captured_requests r
JOIN endpoints e ON r.endpoint_id = e.id
WHERE e.project_id = $1
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetCapturedRequestByID :one
SELECT r.id, r.endpoint_id, r.request_id, r.http_method, r.headers, r.query_params,
       r.raw_body, r.masked_body, r.parsed_json, r.client_ip, r.response_status, r.processing_status, r.created_at,
       e.name as endpoint_name, e.slug as endpoint_slug, e.project_id
FROM captured_requests r
JOIN endpoints e ON r.endpoint_id = e.id
WHERE r.id = $1 LIMIT 1;

-- name: VerifyRequestOwnership :one
SELECT r.id
FROM captured_requests r
JOIN endpoints e ON e.id = r.endpoint_id
JOIN projects p ON p.id = e.project_id
WHERE r.id = $1 AND p.organization_id = $2
LIMIT 1;

-- name: UpdateRequestProcessingStatus :exec
UPDATE captured_requests
SET processing_status = $2
WHERE id = $1;
