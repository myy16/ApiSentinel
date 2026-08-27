-- name: CreateAlertChannel :one
INSERT INTO alert_channels (
    project_id,
    name,
    channel_type,
    webhook_url,
    min_severity,
    is_enabled
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListAlertChannelsByProject :many
SELECT * FROM alert_channels
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetAlertChannelByID :one
SELECT * FROM alert_channels
WHERE id = $1;

-- name: VerifyAlertChannelOwnership :one
SELECT ac.id
FROM alert_channels ac
JOIN projects p ON p.id = ac.project_id
WHERE ac.id = $1 AND p.organization_id = $2
LIMIT 1;

-- name: DeleteAlertChannel :exec
DELETE FROM alert_channels
WHERE id = $1;

-- name: ToggleAlertChannel :one
UPDATE alert_channels
SET is_enabled = $2
WHERE id = $1
RETURNING *;
