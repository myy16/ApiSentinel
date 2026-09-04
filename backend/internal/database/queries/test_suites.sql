-- name: CreateReplayTestSuite :one
INSERT INTO replay_test_suites (
    project_id, name, description, request_ids, target_environment, target_url, renew_idempotency, custom_headers
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListReplayTestSuitesByProject :many
SELECT * FROM replay_test_suites
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetReplayTestSuiteByID :one
SELECT * FROM replay_test_suites
WHERE id = $1;

-- name: DeleteReplayTestSuite :exec
DELETE FROM replay_test_suites
WHERE id = $1 AND project_id = $2;

-- name: CreateReplayTestRun :one
INSERT INTO replay_test_runs (
    suite_id, status, total_steps, passed_steps, failed_steps, total_latency_ms, step_results
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateReplayTestRunResult :one
UPDATE replay_test_runs
SET status = $2,
    passed_steps = $3,
    failed_steps = $4,
    total_latency_ms = $5,
    step_results = $6,
    completed_at = $7
WHERE id = $1
RETURNING *;

-- name: ListReplayTestRunsBySuite :many
SELECT * FROM replay_test_runs
WHERE suite_id = $1
ORDER BY created_at DESC
LIMIT 20;

-- name: VerifyTestSuiteOwnership :one
SELECT ts.id
FROM replay_test_suites ts
JOIN projects p ON p.id = ts.project_id
WHERE ts.id = $1 AND p.organization_id = $2
LIMIT 1;
