-- 002_endpoint_schemas.sql: Create endpoint_schemas table for JSON Schema contract validation

CREATE TABLE IF NOT EXISTS endpoint_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID UNIQUE NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    json_schema JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
