-- 014_ai_opt_in_settings.sql
-- Add Organizational Privacy & AI Opt-in controls (Milestone 14)

ALTER TABLE organizations 
ADD COLUMN IF NOT EXISTS ai_enabled BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS ai_data_sharing_level TEXT NOT NULL DEFAULT 'SANITIZED',
ADD COLUMN IF NOT EXISTS ai_custom_redaction_patterns JSONB NOT NULL DEFAULT '[]'::jsonb;
