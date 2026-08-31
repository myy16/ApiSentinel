-- 1. Add Outbox state machine and worker lease lock columns to forwarding_dlq
ALTER TABLE forwarding_dlq ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ;
ALTER TABLE forwarding_dlq ADD COLUMN IF NOT EXISTS locked_by TEXT;
ALTER TABLE forwarding_dlq ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 3;
ALTER TABLE forwarding_dlq ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_forwarding_dlq_status_retry ON forwarding_dlq(status, next_retry_at) WHERE status IN ('PENDING', 'RETRY_WAIT');
CREATE INDEX IF NOT EXISTS idx_forwarding_dlq_locked ON forwarding_dlq(locked_at) WHERE locked_at IS NOT NULL;
