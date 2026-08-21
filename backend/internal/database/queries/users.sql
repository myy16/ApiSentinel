-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, created_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, created_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateOrganization :one
INSERT INTO organizations (name)
VALUES ($1)
RETURNING id, name, created_at;

-- name: GetOrganizationByID :one
SELECT id, name, created_at
FROM organizations
WHERE id = $1 LIMIT 1;

-- name: CreateMembership :one
INSERT INTO memberships (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, organization_id, user_id, role, created_at;

-- name: GetMembership :one
SELECT id, organization_id, user_id, role, created_at
FROM memberships
WHERE organization_id = $1 AND user_id = $2 LIMIT 1;

-- name: ListUserMemberships :many
SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, o.name as organization_name
FROM memberships m
JOIN organizations o ON m.organization_id = o.id
WHERE m.user_id = $1
ORDER BY m.created_at ASC;
