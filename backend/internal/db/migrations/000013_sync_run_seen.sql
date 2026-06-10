-- +goose Up
CREATE TABLE sync_run_seen (
  run_id uuid NOT NULL REFERENCES sync_runs(id) ON DELETE CASCADE,
  source_patch_id int NOT NULL,
  PRIMARY KEY (run_id, source_patch_id)
);

-- +goose Down
DROP TABLE IF EXISTS sync_run_seen;
