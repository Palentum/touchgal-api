-- +goose Up
DROP INDEX IF EXISTS api_applications_user_unique_idx;
CREATE UNIQUE INDEX IF NOT EXISTS api_applications_user_current_unique_idx ON api_applications(user_id) WHERE status <> 'rejected';

-- +goose Down
DROP INDEX IF EXISTS api_applications_user_current_unique_idx;
CREATE UNIQUE INDEX IF NOT EXISTS api_applications_user_unique_idx ON api_applications(user_id);
