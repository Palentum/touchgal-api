-- +goose Up
CREATE TABLE game_search_ngrams (
  game_unique_id varchar(8) NOT NULL REFERENCES games(unique_id) ON DELETE CASCADE,
  gram text NOT NULL
);

INSERT INTO game_search_ngrams (game_unique_id, gram)
SELECT DISTINCT source.unique_id, grams.gram
FROM (
  SELECT unique_id, lower(search_text) AS search_text, char_length(lower(search_text)) AS search_len
  FROM games
  WHERE search_text <> '' AND deleted_at IS NULL AND content_limit IN ('sfw', 'nsfw')
) AS source
CROSS JOIN LATERAL generate_series(1, source.search_len) AS pos(i)
CROSS JOIN LATERAL (
  VALUES
    (substring(source.search_text FROM pos.i FOR 1)),
    (CASE WHEN pos.i < source.search_len THEN substring(source.search_text FROM pos.i FOR 2) END)
) AS grams(gram)
WHERE grams.gram IS NOT NULL AND grams.gram <> '';

ALTER TABLE game_search_ngrams ADD PRIMARY KEY (game_unique_id, gram);
CREATE INDEX game_search_ngrams_gram_game_idx ON game_search_ngrams(gram, game_unique_id);

-- +goose Down
DROP TABLE IF EXISTS game_search_ngrams;
