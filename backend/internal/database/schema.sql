-- ApiSentinel Database Schema (PostgreSQL 16)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Organizations
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Memberships (Organization <-> User with Role)
CREATE TABLE IF NOT EXISTS memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'OWNER',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT org_user_unique UNIQUE (organization_id, user_id)
);

-- 4. Projects
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Endpoints
CREATE TABLE IF NOT EXISTS endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug VARCHAR(128) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    mode VARCHAR(30) NOT NULL DEFAULT 'PASS',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    upstream_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Captured Requests
CREATE TABLE IF NOT EXISTS captured_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    request_id VARCHAR(100) UNIQUE NOT NULL,
    http_method VARCHAR(10) NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    query_params JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_body TEXT,
    masked_body TEXT,
    parsed_json JSONB,
    client_ip INET,
    response_status INT,
    processing_status VARCHAR(30) NOT NULL DEFAULT 'RECEIVED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for high throughput query performance
CREATE INDEX IF NOT EXISTS idx_captured_requests_endpoint_created ON captured_requests(endpoint_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_captured_requests_request_id ON captured_requests(request_id);

-- 7. Security Rules
CREATE TABLE IF NOT EXISTS rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    category VARCHAR(50) NOT NULL,
    rule_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'HIGH',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    configuration JSONB
);

-- 8. Security Findings
CREATE TABLE IF NOT EXISTS security_findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES captured_requests(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES rules(id) ON DELETE SET NULL,
    category VARCHAR(50) NOT NULL,
    type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    action VARCHAR(20) NOT NULL,
    field_path TEXT,
    message TEXT NOT NULL,
    evidence_masked TEXT,
    confidence DOUBLE PRECISION DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_security_findings_request_id ON security_findings(request_id);

-- 9. Policies
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    configuration JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 10. Mock Rules
CREATE TABLE IF NOT EXISTS mock_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    condition JSONB,
    status_code INT NOT NULL DEFAULT 200,
    delay_ms INT NOT NULL DEFAULT 0,
    response_headers JSONB,
    response_body JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

-- 11. Replay Jobs
CREATE TABLE IF NOT EXISTS replay_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_request_id UUID NOT NULL REFERENCES captured_requests(id) ON DELETE CASCADE,
    target_type VARCHAR(30) NOT NULL,
    target_url TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    response_status INT,
    response_body TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- 12. Local Agents
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    token_hash TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'OFFLINE',
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 13. Alert Channels (Slack, Discord, Telegram, Webhook)
CREATE TABLE IF NOT EXISTS alert_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    channel_type VARCHAR(50) NOT NULL,
    webhook_url TEXT NOT NULL,
    min_severity VARCHAR(20) NOT NULL DEFAULT 'HIGH',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_channels_project_id ON alert_channels(project_id);

-- 14. Forwarding Configurations
CREATE TABLE IF NOT EXISTS forwarding_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID UNIQUE NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    target_url TEXT NOT NULL,
    max_retries INT NOT NULL DEFAULT 3,
    timeout_ms INT NOT NULL DEFAULT 5000,
    custom_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 15. Forwarding Dead Letter Queue (DLQ)
CREATE TABLE IF NOT EXISTS forwarding_dlq (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    request_id UUID NOT NULL REFERENCES captured_requests(id) ON DELETE CASCADE,
    target_url TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    payload TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'FAILED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 16. Endpoint Schemas (Contracts for JSON Schema validation)
CREATE TABLE IF NOT EXISTS endpoint_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID UNIQUE NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    json_schema JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 17. API Keys (Multi-Key Management with Rotation & Revocation)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_project ON api_keys(project_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_lookup ON api_keys(key_prefix, key_hash);

-- 18. Per-endpoint encrypted webhook signature configuration
CREATE TABLE IF NOT EXISTS endpoint_webhook_security (
    endpoint_id UUID PRIMARY KEY REFERENCES endpoints(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    encrypted_secret TEXT NOT NULL,
    require_signature BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT endpoint_webhook_security_provider_check
        CHECK (provider IN ('stripe', 'github', 'shopify', 'generic'))
);

