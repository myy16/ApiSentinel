-- name: CreateMockRule :one
INSERT INTO mock_rules (endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled;

-- name: ListMockRulesByEndpoint :many
SELECT id, endpoint_id, name, condition, status_code, delay_ms, response_headers, response_body, enabled
FROM mock_rules
WHERE endpoint_id = $1
ORDER BY id ASC;

-- name: DeleteMockRule :exec
DELETE FROM mock_rules
WHERE id = $1 AND endpoint_id = $2;
