-- +goose Up
CREATE TABLE api_applications (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  applicant_name varchar(100) NOT NULL,
  project_name varchar(160) NOT NULL DEFAULT '',
  project_url varchar(1000) NOT NULL,
  expected_daily_requests int NOT NULL CHECK (expected_daily_requests > 0),
  usage_scenario text NOT NULL DEFAULT '',
  status varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'revoked')),
  default_minute_limit int NOT NULL DEFAULT 60,
  default_daily_limit int NOT NULL DEFAULT 5000,
  review_note text NOT NULL DEFAULT '',
  reviewed_by uuid REFERENCES users(id),
  reviewed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX api_applications_user_status_idx ON api_applications(user_id, status);
CREATE INDEX api_applications_status_created_idx ON api_applications(status, created_at DESC);
CREATE TRIGGER api_applications_set_updated_at BEFORE UPDATE ON api_applications FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE api_tokens (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  application_id uuid NOT NULL REFERENCES api_applications(id) ON DELETE CASCADE,
  name varchar(100) NOT NULL,
  token_prefix varchar(32) UNIQUE NOT NULL,
  token_hash varchar(128) UNIQUE NOT NULL,
  status varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'revoked')),
  minute_limit int NOT NULL DEFAULT 60,
  daily_limit int NOT NULL DEFAULT 5000,
  last_used_at timestamptz,
  expires_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX api_tokens_user_status_idx ON api_tokens(user_id, status);
CREATE INDEX api_tokens_application_id_idx ON api_tokens(application_id);
CREATE TRIGGER api_tokens_set_updated_at BEFORE UPDATE ON api_tokens FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS api_applications;
