-- +goose Up
ALTER TABLE api_usage_daily ADD COLUMN IF NOT EXISTS total_latency_ms bigint NOT NULL DEFAULT 0;
UPDATE api_usage_daily
SET total_latency_ms = total_requests::bigint * avg_latency_ms
WHERE total_latency_ms = 0 AND total_requests > 0;

ALTER TABLE api_usage_origin_daily ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE api_usage_origin_daily ADD COLUMN IF NOT EXISTS application_id uuid REFERENCES api_applications(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS api_logs_created_idx ON api_request_logs(created_at);
CREATE INDEX IF NOT EXISTS api_usage_daily_user_date_idx ON api_usage_daily(user_id, date DESC);
CREATE INDEX IF NOT EXISTS api_usage_daily_date_idx ON api_usage_daily(date);
CREATE INDEX IF NOT EXISTS api_usage_origin_user_date_idx ON api_usage_origin_daily(user_id, date DESC);
CREATE INDEX IF NOT EXISTS api_usage_origin_date_idx ON api_usage_origin_daily(date);
CREATE INDEX IF NOT EXISTS api_usage_origin_token_date_idx ON api_usage_origin_daily(token_id, date DESC);

CREATE TABLE api_usage_route_daily (
  id bigserial PRIMARY KEY,
  token_id uuid NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  application_id uuid NOT NULL REFERENCES api_applications(id) ON DELETE CASCADE,
  date date NOT NULL,
  route varchar(300) NOT NULL DEFAULT '',
  requests int NOT NULL DEFAULT 0,
  error_requests int NOT NULL DEFAULT 0,
  total_latency_ms bigint NOT NULL DEFAULT 0,
  UNIQUE(token_id, date, route)
);
CREATE INDEX api_usage_route_user_date_idx ON api_usage_route_daily(user_id, date DESC);
CREATE INDEX api_usage_route_date_idx ON api_usage_route_daily(date);
CREATE INDEX api_usage_route_token_date_idx ON api_usage_route_daily(token_id, date DESC);

CREATE TABLE api_usage_ip_daily (
  id bigserial PRIMARY KEY,
  token_id uuid NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  application_id uuid NOT NULL REFERENCES api_applications(id) ON DELETE CASCADE,
  date date NOT NULL,
  ip varchar(100) NOT NULL DEFAULT '',
  UNIQUE(token_id, date, ip)
);
CREATE INDEX api_usage_ip_user_date_idx ON api_usage_ip_daily(user_id, date DESC);
CREATE INDEX api_usage_ip_date_idx ON api_usage_ip_daily(date);
CREATE INDEX api_usage_ip_token_date_idx ON api_usage_ip_daily(token_id, date DESC);

INSERT INTO api_usage_daily (token_id, user_id, application_id, date, total_requests, success_requests, error_requests, total_latency_ms, avg_latency_ms, unique_ips)
SELECT token_id,
       user_id,
       application_id,
       created_at::date,
       count(*)::int,
       count(*) FILTER (WHERE status_code >= 200 AND status_code < 400)::int,
       count(*) FILTER (WHERE status_code >= 400)::int,
       coalesce(sum(latency_ms), 0)::bigint,
       CASE WHEN count(*) = 0 THEN 0 ELSE (coalesce(sum(latency_ms), 0) / count(*))::int END,
       count(DISTINCT nullif(ip, ''))::int
FROM api_request_logs
WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL
GROUP BY token_id, user_id, application_id, created_at::date
ON CONFLICT (token_id, date) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  application_id = EXCLUDED.application_id,
  total_requests = EXCLUDED.total_requests,
  success_requests = EXCLUDED.success_requests,
  error_requests = EXCLUDED.error_requests,
  total_latency_ms = EXCLUDED.total_latency_ms,
  avg_latency_ms = EXCLUDED.avg_latency_ms,
  unique_ips = EXCLUDED.unique_ips,
  updated_at = now();

INSERT INTO api_usage_origin_daily (token_id, user_id, application_id, date, origin, referer_host, ip_prefix, requests)
SELECT token_id,
       user_id,
       application_id,
       date,
       origin,
       referer_host,
       ip_prefix,
       count(*)::int
FROM (
  SELECT token_id,
         user_id,
         application_id,
         created_at::date AS date,
         origin,
         coalesce(nullif(split_part(regexp_replace(referer, '^https?://', ''), '/', 1), ''), 'unknown') AS referer_host,
         '' AS ip_prefix
  FROM api_request_logs
  WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL
) logs
GROUP BY token_id, user_id, application_id, date, origin, referer_host, ip_prefix
ON CONFLICT (token_id, date, origin, referer_host, ip_prefix) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  application_id = EXCLUDED.application_id,
  requests = EXCLUDED.requests;

INSERT INTO api_usage_route_daily (token_id, user_id, application_id, date, route, requests, error_requests, total_latency_ms)
SELECT token_id,
       user_id,
       application_id,
       created_at::date,
       route,
       count(*)::int,
       count(*) FILTER (WHERE status_code >= 400)::int,
       coalesce(sum(latency_ms), 0)::bigint
FROM api_request_logs
WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL
GROUP BY token_id, user_id, application_id, created_at::date, route
ON CONFLICT (token_id, date, route) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  application_id = EXCLUDED.application_id,
  requests = EXCLUDED.requests,
  error_requests = EXCLUDED.error_requests,
  total_latency_ms = EXCLUDED.total_latency_ms;

INSERT INTO api_usage_ip_daily (token_id, user_id, application_id, date, ip)
SELECT DISTINCT token_id,
       user_id,
       application_id,
       created_at::date,
       ip
FROM api_request_logs
WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL AND ip <> ''
ON CONFLICT (token_id, date, ip) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  application_id = EXCLUDED.application_id;

-- +goose Down
DROP TABLE IF EXISTS api_usage_ip_daily;
DROP TABLE IF EXISTS api_usage_route_daily;

DROP INDEX IF EXISTS api_usage_ip_date_idx;
DROP INDEX IF EXISTS api_usage_route_date_idx;
DROP INDEX IF EXISTS api_usage_origin_date_idx;
DROP INDEX IF EXISTS api_usage_daily_date_idx;
DROP INDEX IF EXISTS api_usage_origin_token_date_idx;
DROP INDEX IF EXISTS api_usage_origin_user_date_idx;
DROP INDEX IF EXISTS api_usage_daily_user_date_idx;
DROP INDEX IF EXISTS api_logs_created_idx;

ALTER TABLE api_usage_origin_daily DROP COLUMN IF EXISTS application_id;
ALTER TABLE api_usage_origin_daily DROP COLUMN IF EXISTS user_id;
ALTER TABLE api_usage_daily DROP COLUMN IF EXISTS total_latency_ms;
