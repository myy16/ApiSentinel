-- 015_mock_rules_timestamps.sql
-- Ensure mock_rules table has created_at and updated_at columns for sorting and auditability

ALTER TABLE mock_rules ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE mock_rules ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
