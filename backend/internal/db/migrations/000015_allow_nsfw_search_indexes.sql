-- +goose Up
CREATE INDEX games_public_allow_nsfw_search_text_trgm_idx ON games USING gin(search_text gin_trgm_ops)
  WHERE deleted_at IS NULL AND content_limit IN ('sfw', 'nsfw');
CREATE INDEX games_public_allow_nsfw_name_unique_id_idx ON games(name, unique_id)
  WHERE deleted_at IS NULL AND content_limit IN ('sfw', 'nsfw');

-- +goose Down
DROP INDEX IF EXISTS games_public_allow_nsfw_name_unique_id_idx;
DROP INDEX IF EXISTS games_public_allow_nsfw_search_text_trgm_idx;
