-- +goose Up
DELETE FROM api_tokens
WHERE status <> 'active'
   OR (expires_at IS NOT NULL AND expires_at <= now());

ALTER TABLE api_tokens DROP COLUMN revoked_at;

ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_status_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_status_check CHECK (status = 'active');

-- +goose Down
ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_status_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_status_check CHECK (status IN ('active', 'disabled', 'revoked'));
ALTER TABLE api_tokens ADD COLUMN revoked_at timestamptz;
