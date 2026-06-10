-- +goose Up
CREATE INDEX games_public_search_text_trgm_idx ON games USING gin(search_text gin_trgm_ops)
  WHERE deleted_at IS NULL AND content_limit = 'sfw';
CREATE INDEX games_public_name_unique_id_idx ON games(name, unique_id)
  WHERE deleted_at IS NULL AND content_limit = 'sfw';

DROP INDEX IF EXISTS games_search_text_trgm_idx;
DROP INDEX IF EXISTS games_name_trgm_idx;
DROP INDEX IF EXISTS game_aliases_name_trgm_idx;

-- +goose Down
CREATE INDEX games_search_text_trgm_idx ON games USING gin(search_text gin_trgm_ops);
CREATE INDEX games_name_trgm_idx ON games USING gin(name gin_trgm_ops);
CREATE INDEX game_aliases_name_trgm_idx ON game_aliases USING gin(name gin_trgm_ops);

DROP INDEX IF EXISTS games_public_name_unique_id_idx;
DROP INDEX IF EXISTS games_public_search_text_trgm_idx;
