-- 011_replay_target_override.sql: Replay target environment, custom headers, and latency tracking

ALTER TABLE replay_jobs
ADD COLUMN IF NOT EXISTS environment VARCHAR(50) NOT NULL DEFAULT 'CUSTOM',
ADD COLUMN IF NOT EXISTS latency_ms INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS custom_headers JSONB NOT NULL DEFAULT '{}'::jsonb;
