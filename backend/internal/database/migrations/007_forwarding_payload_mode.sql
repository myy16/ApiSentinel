-- 1. Add payload_mode to forwarding_configs (REDACTED default, RAW optional)
ALTER TABLE forwarding_configs ADD COLUMN IF NOT EXISTS payload_mode VARCHAR(20) NOT NULL DEFAULT 'REDACTED';

-- 2. Add payload_mode to forwarding_dlq
ALTER TABLE forwarding_dlq ADD COLUMN IF NOT EXISTS payload_mode VARCHAR(20) NOT NULL DEFAULT 'REDACTED';
