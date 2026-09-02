-- 010_schema_drift_events.sql: Store schema drift incidents and difference analysis

CREATE TABLE IF NOT EXISTS schema_drift_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    schema_baseline_id UUID NOT NULL REFERENCES schema_baselines(id) ON DELETE CASCADE,
    request_id UUID REFERENCES captured_requests(id) ON DELETE CASCADE,
    drift_type VARCHAR(50) NOT NULL,
    diff_json JSONB NOT NULL,
    is_acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schema_drift_events_endpoint ON schema_drift_events (endpoint_id, is_acknowledged, created_at DESC);
