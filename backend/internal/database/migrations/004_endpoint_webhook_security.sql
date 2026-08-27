-- Per-endpoint webhook signature configuration. The secret is encrypted by the application
-- before it reaches this table and is never returned by the API.
CREATE TABLE IF NOT EXISTS endpoint_webhook_security (
    endpoint_id UUID PRIMARY KEY REFERENCES endpoints(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    encrypted_secret TEXT NOT NULL,
    require_signature BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT endpoint_webhook_security_provider_check
        CHECK (provider IN ('stripe', 'github', 'shopify', 'generic'))
);
