-- 013_endpoint_traffic_controls.sql: Add payload size limits and rate limiting / spike protection configuration

ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS max_payload_size_bytes INT NOT NULL DEFAULT 5242880,
ADD COLUMN IF NOT EXISTS rate_limit_rpm INT NOT NULL DEFAULT 120,
ADD COLUMN IF NOT EXISTS burst_threshold INT NOT NULL DEFAULT 30;
