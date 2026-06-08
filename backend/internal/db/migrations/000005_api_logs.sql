-- +goose Up
CREATE TABLE api_request_logs (
  id bigserial PRIMARY KEY,
  token_id uuid REFERENCES api_tokens(id) ON DELETE SET NULL,
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  application_id uuid REFERENCES api_applications(id) ON DELETE SET NULL,
  method varchar(20) NOT NULL,
  path varchar(500) NOT NULL,
  route varchar(300) NOT NULL,
  status_code int NOT NULL,
  latency_ms int NOT NULL DEFAULT 0,
  ip varchar(100) NOT NULL DEFAULT '',
  user_agent varchar(1000) NOT NULL DEFAULT '',
  origin varchar(1000) NOT NULL DEFAULT '',
  referer varchar(1000) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX api_logs_token_created_idx ON api_request_logs(token_id, created_at DESC);
CREATE INDEX api_logs_user_created_idx ON api_request_logs(user_id, created_at DESC);
CREATE INDEX api_logs_application_created_idx ON api_request_logs(application_id, created_at DESC);
CREATE INDEX api_logs_route_created_idx ON api_request_logs(route, created_at DESC);
CREATE INDEX api_logs_origin_created_idx ON api_request_logs(origin, created_at DESC);

CREATE TABLE api_usage_daily (
  id bigserial PRIMARY KEY,
  token_id uuid REFERENCES api_tokens(id) ON DELETE CASCADE,
  user_id uuid REFERENCES users(id) ON DELETE CASCADE,
  application_id uuid REFERENCES api_applications(id) ON DELETE CASCADE,
  date date NOT NULL,
  total_requests int NOT NULL DEFAULT 0,
  success_requests int NOT NULL DEFAULT 0,
  error_requests int NOT NULL DEFAULT 0,
  avg_latency_ms int NOT NULL DEFAULT 0,
  unique_ips int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(token_id, date)
);
CREATE TRIGGER api_usage_daily_set_updated_at BEFORE UPDATE ON api_usage_daily FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE api_usage_origin_daily (
  id bigserial PRIMARY KEY,
  token_id uuid REFERENCES api_tokens(id) ON DELETE CASCADE,
  date date NOT NULL,
  origin varchar(1000) NOT NULL DEFAULT '',
  referer_host varchar(500) NOT NULL DEFAULT '',
  ip_prefix varchar(100) NOT NULL DEFAULT '',
  requests int NOT NULL DEFAULT 0,
  UNIQUE(token_id, date, origin, referer_host, ip_prefix)
);

-- +goose Down
DROP TABLE IF EXISTS api_usage_origin_daily;
DROP TABLE IF EXISTS api_usage_daily;
DROP TABLE IF EXISTS api_request_logs;
