-- name: UpsertEndpointWebhookSecurity :one
INSERT INTO endpoint_webhook_security (endpoint_id, provider, encrypted_secret, require_signature)
VALUES ($1, $2, $3, $4)
ON CONFLICT (endpoint_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    encrypted_secret = EXCLUDED.encrypted_secret,
    require_signature = EXCLUDED.require_signature,
    updated_at = NOW()
RETURNING endpoint_id, provider, encrypted_secret, require_signature, created_at, updated_at;

-- name: GetEndpointWebhookSecurity :one
SELECT endpoint_id, provider, encrypted_secret, require_signature, created_at, updated_at
FROM endpoint_webhook_security
WHERE endpoint_id = $1;

-- name: DeleteEndpointWebhookSecurity :exec
DELETE FROM endpoint_webhook_security
WHERE endpoint_id = $1;
