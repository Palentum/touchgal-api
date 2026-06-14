-- +goose Up
CREATE TABLE game_resources (
  source_resource_id int PRIMARY KEY,
  game_unique_id varchar(8) NOT NULL REFERENCES games(unique_id) ON DELETE CASCADE,
  name varchar(300) NOT NULL DEFAULT '',
  introduction text NOT NULL DEFAULT '',
  categories text[] NOT NULL DEFAULT '{}',
  resource_type varchar(20) NOT NULL CHECK (resource_type IN ('resource', 'patch')),
  sizes text[] NOT NULL DEFAULT '{}',
  published_at timestamptz NOT NULL,
  source_updated_at timestamptz NOT NULL,
  synced_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX game_resources_game_unique_id_published_at_idx ON game_resources(game_unique_id, published_at DESC, source_resource_id DESC);
CREATE INDEX game_resources_source_updated_at_idx ON game_resources(source_updated_at);

-- +goose Down
DROP TABLE IF EXISTS game_resources;
