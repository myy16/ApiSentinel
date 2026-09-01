-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    organization_id,
    project_id,
    user_id,
    action,
    resource_type,
    resource_id,
    justification,
    ip_address,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListAuditLogsByOrganization :many
SELECT * FROM audit_logs
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByProject :many
SELECT * FROM audit_logs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
