-- +goose Up
CREATE INDEX users_admin_status_created_at_idx ON users(is_admin, status, created_at DESC) INCLUDE (email);

-- +goose Down
DROP INDEX IF EXISTS users_admin_status_created_at_idx;
