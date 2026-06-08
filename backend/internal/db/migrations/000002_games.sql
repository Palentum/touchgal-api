-- +goose Up
CREATE TABLE games (
  unique_id varchar(8) PRIMARY KEY,
  source_patch_id int NOT NULL,
  name varchar(1007) NOT NULL,
  introduction text NOT NULL DEFAULT '',
  banner_url varchar(1007) NOT NULL DEFAULT '',
  released varchar(107) NOT NULL DEFAULT 'unknown',
  content_limit varchar(107) NOT NULL DEFAULT '',
  types text[] NOT NULL DEFAULT '{}',
  languages text[] NOT NULL DEFAULT '{}',
  platforms text[] NOT NULL DEFAULT '{}',
  source_created_at timestamptz NOT NULL,
  source_updated_at timestamptz NOT NULL,
  resource_updated_at timestamptz NOT NULL,
  search_text text NOT NULL DEFAULT '',
  synced_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE UNIQUE INDEX games_unique_id_uidx ON games(unique_id);
CREATE INDEX games_source_patch_id_idx ON games(source_patch_id);
CREATE INDEX games_source_updated_at_idx ON games(source_updated_at);
CREATE INDEX games_resource_updated_at_idx ON games(resource_updated_at);
CREATE INDEX games_content_limit_idx ON games(content_limit);
CREATE INDEX games_search_text_trgm_idx ON games USING gin(search_text gin_trgm_ops);
CREATE INDEX games_name_trgm_idx ON games USING gin(name gin_trgm_ops);

CREATE TABLE game_aliases (
  id bigserial PRIMARY KEY,
  game_unique_id varchar(8) NOT NULL REFERENCES games(unique_id) ON DELETE CASCADE,
  name varchar(1007) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(game_unique_id, name)
);
CREATE INDEX game_aliases_name_trgm_idx ON game_aliases USING gin(name gin_trgm_ops);

CREATE TABLE tags (
  id bigserial PRIMARY KEY,
  name varchar(107) UNIQUE NOT NULL,
  aliases text[] NOT NULL DEFAULT '{}',
  source varchar(50) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER tags_set_updated_at BEFORE UPDATE ON tags FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE game_tags (
  game_unique_id varchar(8) NOT NULL REFERENCES games(unique_id) ON DELETE CASCADE,
  tag_id bigint NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY(game_unique_id, tag_id)
);

CREATE TABLE companies (
  id bigserial PRIMARY KEY,
  name varchar(107) UNIQUE NOT NULL,
  aliases text[] NOT NULL DEFAULT '{}',
  official_websites text[] NOT NULL DEFAULT '{}',
  primary_languages text[] NOT NULL DEFAULT '{}',
  parent_brands text[] NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER companies_set_updated_at BEFORE UPDATE ON companies FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE game_companies (
  game_unique_id varchar(8) NOT NULL REFERENCES games(unique_id) ON DELETE CASCADE,
  company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  PRIMARY KEY(game_unique_id, company_id)
);

CREATE TABLE game_rating_stats (
  game_unique_id varchar(8) PRIMARY KEY REFERENCES games(unique_id) ON DELETE CASCADE,
  average_overall numeric(4,1) NOT NULL DEFAULT 0,
  count int NOT NULL DEFAULT 0,
  rec_strong_no int NOT NULL DEFAULT 0,
  rec_no int NOT NULL DEFAULT 0,
  rec_neutral int NOT NULL DEFAULT 0,
  rec_yes int NOT NULL DEFAULT 0,
  rec_strong_yes int NOT NULL DEFAULT 0,
  histogram jsonb NOT NULL DEFAULT '{}'::jsonb,
  synced_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS game_rating_stats;
DROP TABLE IF EXISTS game_companies;
DROP TABLE IF EXISTS companies;
DROP TABLE IF EXISTS game_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS game_aliases;
DROP TABLE IF EXISTS games;
