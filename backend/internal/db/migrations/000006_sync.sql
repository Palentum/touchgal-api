-- +goose Up
CREATE TABLE sync_runs (
  id uuid PRIMARY KEY,
  mode varchar(20) NOT NULL CHECK (mode IN ('incremental', 'full')),
  status varchar(20) NOT NULL CHECK (status IN ('running', 'success', 'failed')),
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  source_max_updated_at timestamptz,
  games_seen int NOT NULL DEFAULT 0,
  games_upserted int NOT NULL DEFAULT 0,
  games_deleted int NOT NULL DEFAULT 0,
  error_message text NOT NULL DEFAULT ''
);
CREATE INDEX sync_runs_started_idx ON sync_runs(started_at DESC);
CREATE INDEX sync_runs_status_idx ON sync_runs(status);

-- +goose Down
DROP TABLE IF EXISTS sync_runs;
