-- +goose Up
ALTER TABLE users
  ADD COLUMN minute_limit int NOT NULL DEFAULT 60 CHECK (minute_limit > 0),
  ADD COLUMN daily_limit int NOT NULL DEFAULT 5000 CHECK (daily_limit > 0),
  ADD CONSTRAINT users_rate_limits_order_check CHECK (daily_limit >= minute_limit);

-- +goose Down
ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_rate_limits_order_check,
  DROP COLUMN IF EXISTS daily_limit,
  DROP COLUMN IF EXISTS minute_limit;
