-- name: CreateAPIKey :one
INSERT INTO api_keys (project_id, name, key_prefix, key_hash, created_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, name, key_prefix, created_by, expires_at, last_used_at, revoked_at, created_at;

-- name: ListAPIKeysByProject :many
SELECT id, project_id, name, key_prefix, created_by, expires_at, last_used_at, revoked_at, created_at
FROM api_keys
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetAPIKeyByPrefixAndHash :one
SELECT id, project_id, name, key_prefix, key_hash, created_by, expires_at, last_used_at, revoked_at, created_at
FROM api_keys
WHERE key_prefix = $1 AND key_hash = $2 AND revoked_at IS NULL
LIMIT 1;

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = NOW()
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, name, key_prefix, revoked_at;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = $1 AND project_id = $2;
