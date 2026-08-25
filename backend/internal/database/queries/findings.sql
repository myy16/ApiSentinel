-- name: CreateSecurityFinding :one
INSERT INTO security_findings (
    request_id, rule_id, category, type, severity,
    action, field_path, message, evidence_masked, confidence
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: ListFindingsByProject :many
SELECT sf.id, sf.request_id, sf.rule_id, sf.category, sf.type, sf.severity,
       sf.action, sf.field_path, sf.message, sf.evidence_masked, sf.confidence, sf.created_at,
       cr.request_id as req_display_id, e.name as endpoint_name
FROM security_findings sf
JOIN captured_requests cr ON sf.request_id = cr.id
JOIN endpoints e ON cr.endpoint_id = e.id
WHERE e.project_id = $1
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
JOIN captured_requests cr ON sf.request_id = cr.id
JOIN endpoints e ON cr.endpoint_id = e.id
WHERE e.project_id = $1;
