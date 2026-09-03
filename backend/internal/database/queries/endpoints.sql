-- name: CreateEndpoint :one
INSERT INTO endpoints (project_id, slug, name, mode, upstream_url, is_active, max_payload_size_bytes, rate_limit_rpm, burst_threshold)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, project_id, slug, name, mode, is_active, upstream_url, max_payload_size_bytes, rate_limit_rpm, burst_threshold, created_at;

-- name: ListEndpointsByProject :many
SELECT e.id, e.project_id, e.slug, e.name, e.mode, e.is_active, e.upstream_url,
       e.max_payload_size_bytes, e.rate_limit_rpm, e.burst_threshold, e.created_at,
       COUNT(r.id)::bigint as request_count
FROM endpoints e
LEFT JOIN captured_requests r ON e.id = r.endpoint_id
WHERE e.project_id = $1
GROUP BY e.id
ORDER BY e.created_at DESC;

-- name: GetEndpointByID :one
SELECT id, project_id, slug, name, mode, is_active, upstream_url, max_payload_size_bytes, rate_limit_rpm, burst_threshold, created_at
FROM endpoints
WHERE id = $1 AND project_id = $2 LIMIT 1;

-- name: GetEndpointBySlug :one
SELECT id, project_id, slug, name, mode, is_active, upstream_url, max_payload_size_bytes, rate_limit_rpm, burst_threshold, created_at
FROM endpoints
WHERE slug = $1 LIMIT 1;

-- name: UpdateEndpoint :one
UPDATE endpoints
SET name = COALESCE($3, name),
    mode = COALESCE($4, mode),
    is_active = COALESCE($5, is_active),
    upstream_url = COALESCE($6, upstream_url),
    max_payload_size_bytes = COALESCE($7, max_payload_size_bytes),
    rate_limit_rpm = COALESCE($8, rate_limit_rpm),
    burst_threshold = COALESCE($9, burst_threshold)
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, slug, name, mode, is_active, upstream_url, max_payload_size_bytes, rate_limit_rpm, burst_threshold, created_at;

-- name: DeleteEndpoint :exec
DELETE FROM endpoints
WHERE id = $1 AND project_id = $2;

-- name: GetEndpointWithOwnership :one
SELECT e.id, e.project_id, p.organization_id
FROM endpoints e
JOIN projects p ON e.project_id = p.id
WHERE e.id = $1 AND p.organization_id = $2
LIMIT 1;

-- name: UpsertEndpointSchema :one
INSERT INTO endpoint_schemas (endpoint_id, json_schema)
VALUES ($1, $2)
ON CONFLICT (endpoint_id) DO UPDATE SET
    json_schema = EXCLUDED.json_schema,
    updated_at = NOW()
RETURNING id, endpoint_id, json_schema, created_at, updated_at;

-- name: GetEndpointSchema :one
SELECT id, endpoint_id, json_schema, created_at, updated_at
FROM endpoint_schemas
WHERE endpoint_id = $1 LIMIT 1;

-- name: DeleteEndpointSchema :exec
DELETE FROM endpoint_schemas
WHERE endpoint_id = $1;

-- name: GetEndpointByIDOnly :one
SELECT id, project_id, slug, name, mode, is_active, upstream_url, max_payload_size_bytes, rate_limit_rpm, burst_threshold, created_at
FROM endpoints
WHERE id = $1 LIMIT 1;
