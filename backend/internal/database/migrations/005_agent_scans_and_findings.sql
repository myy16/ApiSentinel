-- 1. Agent Scans table to track Git / CLI scans
CREATE TABLE IF NOT EXISTS agent_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    branch TEXT NOT NULL DEFAULT '',
    commit_hash TEXT NOT NULL DEFAULT '',
    scan_type TEXT NOT NULL DEFAULT 'STAGED',
    total_findings INT NOT NULL DEFAULT 0,
    action TEXT NOT NULL DEFAULT 'ALLOW',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_scans_project_id ON agent_scans(project_id, created_at DESC);

-- 2. Make request_id nullable on security_findings to accommodate Git/CLI scans
ALTER TABLE security_findings ALTER COLUMN request_id DROP NOT NULL;

-- 3. Add Agent scan / Git context columns to security_findings
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS scan_id UUID REFERENCES agent_scans(id) ON DELETE CASCADE;
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS source_type VARCHAR(30) NOT NULL DEFAULT 'WEBHOOK';
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS file_path TEXT;
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS line_number INT;
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS commit_hash TEXT;
ALTER TABLE security_findings ADD COLUMN IF NOT EXISTS repository TEXT;

CREATE INDEX IF NOT EXISTS idx_security_findings_scan_id ON security_findings(scan_id);
CREATE INDEX IF NOT EXISTS idx_security_findings_project_id ON security_findings(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_findings_source_type ON security_findings(source_type);
