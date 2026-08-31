-- name: CreateSecurityFinding :one
INSERT INTO security_findings (
    request_id, project_id, scan_id, source_type, rule_id, category, type, severity,
    action, field_path, file_path, line_number, commit_hash, repository,
    message, evidence_masked, confidence
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
RETURNING *;

-- name: CreateAgentScan :one
INSERT INTO agent_scans (
    project_id, agent_id, repository, branch, commit_hash, scan_type, total_findings, action
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ListAgentScansByProject :many
SELECT id, project_id, agent_id, repository, branch, commit_hash, scan_type, total_findings, action, created_at
FROM agent_scans
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListFindingsByProject :many
SELECT sf.id, sf.request_id, sf.project_id, sf.scan_id, sf.source_type, sf.rule_id, sf.category, sf.type, sf.severity,
       sf.action, sf.field_path, sf.file_path, sf.line_number, sf.commit_hash, sf.repository,
       sf.message, sf.evidence_masked, sf.confidence, sf.created_at,
       COALESCE(cr.request_id, '') as req_display_id,
       COALESCE(e.name, '') as endpoint_name
FROM security_findings sf
LEFT JOIN captured_requests cr ON sf.request_id = cr.id
LEFT JOIN endpoints e ON cr.endpoint_id = e.id
LEFT JOIN agent_scans ag ON sf.scan_id = ag.id
WHERE (e.project_id = $1 OR sf.project_id = $1 OR ag.project_id = $1)
ORDER BY sf.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetFindingStats :one
SELECT
    COUNT(*) FILTER (WHERE sf.severity = 'CRITICAL')::bigint as critical_count,
    COUNT(*) FILTER (WHERE sf.severity = 'HIGH')::bigint as high_count,
    COUNT(*) FILTER (WHERE sf.severity = 'MEDIUM')::bigint as medium_count,
    COUNT(*) FILTER (WHERE sf.severity = 'INFO')::bigint as info_count,
    COUNT(*)::bigint as total_count
FROM security_findings sf
LEFT JOIN captured_requests cr ON sf.request_id = cr.id
LEFT JOIN endpoints e ON cr.endpoint_id = e.id
LEFT JOIN agent_scans ag ON sf.scan_id = ag.id
WHERE (e.project_id = $1 OR sf.project_id = $1 OR ag.project_id = $1);

