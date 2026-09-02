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
