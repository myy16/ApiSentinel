-- 1. Add request_state to captured_requests if not present
ALTER TABLE captured_requests ADD COLUMN IF NOT EXISTS request_state VARCHAR(30) NOT NULL DEFAULT 'RECEIVED';
CREATE INDEX IF NOT EXISTS idx_captured_requests_request_state ON captured_requests(request_state);

-- 2. Delivery Jobs Table (Core Transactional Outbox)
CREATE TABLE IF NOT EXISTS delivery_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    request_id UUID NOT NULL REFERENCES captured_requests(id) ON DELETE CASCADE,
    target_url TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    attempts INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    locked_by TEXT,
    idempotency_key TEXT,
    last_error TEXT,
    payload_mode VARCHAR(20) NOT NULL DEFAULT 'REDACTED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Indexes for performance & concurrency
CREATE INDEX IF NOT EXISTS idx_delivery_jobs_endpoint_status ON delivery_jobs(endpoint_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_delivery_jobs_status_retry ON delivery_jobs(status, next_retry_at) WHERE status IN ('PENDING', 'RETRY_WAIT');
CREATE INDEX IF NOT EXISTS idx_delivery_jobs_locked ON delivery_jobs(locked_at) WHERE locked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_delivery_jobs_request_id ON delivery_jobs(request_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_delivery_jobs_endpoint_idempotency ON delivery_jobs(endpoint_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- 3. Delivery Attempts Table (Detailed Per-Attempt Telemetry)
CREATE TABLE IF NOT EXISTS delivery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES delivery_jobs(id) ON DELETE CASCADE,
    attempt_number INT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    latency_ms INT NOT NULL DEFAULT 0,
    response_status_code INT,
    error_type VARCHAR(50),
    error_message TEXT,
    request_headers_sent JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_headers_received JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_body_snippet TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_job_id ON delivery_attempts(job_id, attempt_number ASC);

-- 4. Audit Logs Table (Security, Replay & Configuration Audit Trail)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id TEXT NOT NULL,
    justification TEXT,
    ip_address VARCHAR(45),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_project_created ON audit_logs(project_id, created_at DESC);
