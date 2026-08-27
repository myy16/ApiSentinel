-- name: CreateProject :one
INSERT INTO projects (organization_id, name)
VALUES ($1, $2)
RETURNING id, organization_id, name, created_at;

-- name: ListProjectsByOrg :many
SELECT id, organization_id, name, created_at
FROM projects
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: GetProjectByID :one
SELECT id, organization_id, name, created_at
FROM projects
WHERE id = $1 AND organization_id = $2 LIMIT 1;

-- name: UpdateProject :one
UPDATE projects
SET name = $3
WHERE id = $1 AND organization_id = $2
RETURNING id, organization_id, name, created_at;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1 AND organization_id = $2;

-- name: VerifyProjectOwnership :one
SELECT id FROM projects
WHERE id = $1 AND organization_id = $2
LIMIT 1;

-- name: GetProjectOrganizationID :one
SELECT organization_id
FROM projects
WHERE id = $1
LIMIT 1;
