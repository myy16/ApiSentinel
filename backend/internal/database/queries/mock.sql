-- name: CreateMockRule :one
INSERT INTO mock_rules (endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled;

-- name: ListMockRulesByEndpoint :many
SELECT id, endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled
FROM mock_rules
WHERE endpoint_id = $1
ORDER BY enabled DESC, created_at DESC;

-- name: DeleteMockRule :exec
DELETE FROM mock_rules
WHERE id = $1 AND endpoint_id = $2;

-- name: ToggleMockRule :one
UPDATE mock_rules
SET enabled = NOT enabled
WHERE id = $1 AND endpoint_id = $2
RETURNING id, endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled;

-- name: GetMatchingMockRule :one
SELECT id, endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled
FROM mock_rules
WHERE endpoint_id = $1 AND enabled = true
ORDER BY created_at DESC
LIMIT 1;
