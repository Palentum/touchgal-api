-- +goose Up
CREATE UNIQUE INDEX api_applications_user_unique_idx ON api_applications(user_id);

-- +goose Down
DROP INDEX IF EXISTS api_applications_user_unique_idx;
