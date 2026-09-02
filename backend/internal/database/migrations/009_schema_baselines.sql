-- 009_schema_baselines.sql: Versioned schema baselines and OpenAPI import support

CREATE TABLE IF NOT EXISTS schema_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    schema_json JSONB NOT NULL,
    source TEXT NOT NULL DEFAULT 'MANUAL', -- 'OPENAPI', 'INFERRED_PAYLOAD', 'MANUAL'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_endpoint_version UNIQUE (endpoint_id, version)
);

CREATE INDEX IF NOT EXISTS idx_schema_baselines_endpoint ON schema_baselines (endpoint_id, is_active);
