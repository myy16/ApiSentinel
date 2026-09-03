-- 012_replay_test_suites.sql: Replay Test Suites and Scenario Execution Runs

CREATE TABLE IF NOT EXISTS replay_test_suites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    request_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    target_environment VARCHAR(50) NOT NULL DEFAULT 'STAGING',
    target_url TEXT,
    renew_idempotency BOOLEAN NOT NULL DEFAULT TRUE,
    custom_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_replay_test_suites_project ON replay_test_suites (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS replay_test_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES replay_test_suites(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'RUNNING',
    total_steps INT NOT NULL DEFAULT 0,
    passed_steps INT NOT NULL DEFAULT 0,
    failed_steps INT NOT NULL DEFAULT 0,
    total_latency_ms INT NOT NULL DEFAULT 0,
    step_results JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_replay_test_runs_suite ON replay_test_runs (suite_id, created_at DESC);
