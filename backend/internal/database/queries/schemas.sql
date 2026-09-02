-- name: ListSchemaBaselinesByEndpoint :many
SELECT * FROM schema_baselines
WHERE endpoint_id = $1
ORDER BY version DESC;

-- name: GetActiveSchemaBaseline :one
SELECT * FROM schema_baselines
WHERE endpoint_id = $1 AND is_active = TRUE
LIMIT 1;

-- name: GetNextSchemaVersion :one
SELECT COALESCE(MAX(version), 0)::int + 1 AS next_version
FROM schema_baselines
WHERE endpoint_id = $1;

-- name: DeactivateAllSchemaBaselines :exec
UPDATE schema_baselines
SET is_active = FALSE, updated_at = NOW()
WHERE endpoint_id = $1;

-- name: CreateSchemaBaseline :one
INSERT INTO schema_baselines (
    endpoint_id, version, schema_json, source, is_active
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ActivateSchemaBaseline :one
UPDATE schema_baselines
SET is_active = TRUE, updated_at = NOW()
WHERE id = $1 AND endpoint_id = $2
RETURNING *;

-- name: CreateSchemaDriftEvent :one
INSERT INTO schema_drift_events (
    endpoint_id, schema_baseline_id, request_id, drift_type, diff_json, is_acknowledged
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: ListSchemaDriftsByEndpoint :many
SELECT d.*, r.request_id as monotonic_request_id, r.created_at as request_created_at
FROM schema_drift_events d
LEFT JOIN captured_requests r ON d.request_id = r.id
WHERE d.endpoint_id = $1
ORDER BY d.created_at DESC
LIMIT 50;

-- name: AcknowledgeSchemaDrift :one
UPDATE schema_drift_events
SET is_acknowledged = TRUE
WHERE id = $1 AND endpoint_id = $2
RETURNING *;

