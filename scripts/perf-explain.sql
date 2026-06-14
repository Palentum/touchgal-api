\if :{?keyword}
\else
\set keyword summer
\endif
\if :{?page}
\else
\set page 1
\endif
\if :{?limit}
\else
\set limit 20
\endif
\if :{?days}
\else
\set days 30
\endif
\if :{?unique_id}
\else
\set unique_id ''
\endif
\if :{?user_id}
\else
\set user_id ''
\endif

BEGIN READ ONLY;
SET LOCAL statement_timeout = '30s';

\echo 'clean-db search page query'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH patterns AS (
  SELECT :'keyword'::text AS keyword,
         replace(replace(replace(:'keyword'::text, E'\\', E'\\\\'), '%', E'\\%'), '_', E'\\_') AS exact_pattern
), search_patterns AS (
  SELECT keyword,
         exact_pattern,
         exact_pattern || '%' AS prefix_pattern,
         '%' || exact_pattern || '%' AS contains_pattern
  FROM patterns
), ranked_games AS (
  SELECT g.unique_id,
         g.name,
         CASE
           WHEN g.name ILIKE p.exact_pattern ESCAPE E'\\' THEN 3 + similarity(g.name, p.keyword)
           WHEN g.name ILIKE p.prefix_pattern ESCAPE E'\\' THEN 2 + similarity(g.name, p.keyword)
           WHEN g.name ILIKE p.contains_pattern ESCAPE E'\\' THEN 1 + similarity(g.name, p.keyword)
           ELSE 0
         END AS title_rank,
         CASE
           WHEN g.name ILIKE p.contains_pattern ESCAPE E'\\' THEN 0
           ELSE COALESCE((
             SELECT max(CASE
               WHEN a.name ILIKE p.exact_pattern ESCAPE E'\\' THEN 3 + similarity(a.name, p.keyword)
               WHEN a.name ILIKE p.prefix_pattern ESCAPE E'\\' THEN 2 + similarity(a.name, p.keyword)
               WHEN a.name ILIKE p.contains_pattern ESCAPE E'\\' THEN 1 + similarity(a.name, p.keyword)
               ELSE 0
             END)
             FROM game_aliases a
             WHERE a.game_unique_id = g.unique_id
               AND a.name ILIKE p.contains_pattern ESCAPE E'\\'
           ), 0)
         END AS alias_rank,
         similarity(g.search_text, p.keyword) AS metadata_rank
  FROM search_patterns p
  CROSS JOIN games g
  WHERE g.deleted_at IS NULL
    AND g.content_limit = 'sfw'
    AND g.search_text ILIKE p.contains_pattern ESCAPE E'\\'
)
SELECT unique_id, name
FROM ranked_games
ORDER BY
  CASE
    WHEN title_rank > 0 THEN 0
    WHEN alias_rank > 0 THEN 1
    ELSE 2
  END ASC,
  CASE
    WHEN title_rank > 0 THEN title_rank
    WHEN alias_rank > 0 THEN alias_rank
    ELSE metadata_rank
  END DESC,
  name ASC, unique_id ASC
LIMIT (:'limit')::int OFFSET (((:'page')::int - 1) * (:'limit')::int);

\echo 'clean-db search count query'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH patterns AS (
  SELECT replace(replace(replace(:'keyword'::text, E'\\', E'\\\\'), '%', E'\\%'), '_', E'\\_') AS exact_pattern
), search_patterns AS (
  SELECT '%' || exact_pattern || '%' AS contains_pattern
  FROM patterns
)
SELECT count(*)
FROM search_patterns p
CROSS JOIN games g
WHERE g.deleted_at IS NULL
  AND g.content_limit = 'sfw'
  AND g.search_text ILIKE p.contains_pattern ESCAPE E'\\';

\echo 'clean-db game detail primary query'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH selected_game AS (
  SELECT coalesce(nullif(:'unique_id', ''), (
    SELECT unique_id
    FROM games
    WHERE deleted_at IS NULL AND content_limit = 'sfw'
    ORDER BY unique_id
    LIMIT 1
  )) AS unique_id
)
SELECT g.unique_id, g.name, g.introduction, g.banner_url, g.types, g.platforms, g.languages,
       g.source_created_at, g.released, g.source_updated_at, g.resource_updated_at,
       coalesce(r.average_overall, 0), coalesce(r.count, 0), coalesce(r.rec_strong_no, 0), coalesce(r.rec_no, 0),
       coalesce(r.rec_neutral, 0), coalesce(r.rec_yes, 0), coalesce(r.rec_strong_yes, 0)
FROM games g
LEFT JOIN game_rating_stats r ON r.game_unique_id = g.unique_id
JOIN selected_game sg ON sg.unique_id = g.unique_id
WHERE g.deleted_at IS NULL AND g.content_limit = 'sfw';

\echo 'clean-db game detail relation queries'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH selected_game AS (
  SELECT coalesce(nullif(:'unique_id', ''), (
    SELECT unique_id
    FROM games
    WHERE deleted_at IS NULL AND content_limit = 'sfw'
    ORDER BY unique_id
    LIMIT 1
  )) AS unique_id
)
SELECT t.name
FROM selected_game sg
JOIN game_tags gt ON gt.game_unique_id = sg.unique_id
JOIN tags t ON t.id = gt.tag_id
ORDER BY t.name;

EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH selected_game AS (
  SELECT coalesce(nullif(:'unique_id', ''), (
    SELECT unique_id
    FROM games
    WHERE deleted_at IS NULL AND content_limit = 'sfw'
    ORDER BY unique_id
    LIMIT 1
  )) AS unique_id
)
SELECT c.name, c.aliases
FROM selected_game sg
JOIN game_companies gc ON gc.game_unique_id = sg.unique_id
JOIN companies c ON c.id = gc.company_id
ORDER BY c.name;

\echo 'clean-db dashboard aggregate query'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH selected_user AS (
  SELECT coalesce(nullif(:'user_id', '')::uuid, (
    SELECT user_id
    FROM api_usage_daily
    ORDER BY date DESC
    LIMIT 1
  )) AS user_id
),
daily_rollups AS (
  SELECT d.date, d.total_requests, d.success_requests, d.error_requests, d.total_latency_ms
  FROM api_usage_daily d
  JOIN selected_user u ON u.user_id = d.user_id
  WHERE d.date >= (CURRENT_DATE - (((:'days')::int - 1) * INTERVAL '1 day'))::date
),
origin_rollups AS (
  SELECT o.origin, o.referer_host, o.requests
  FROM api_usage_origin_daily o
  JOIN selected_user u ON u.user_id = o.user_id
  WHERE o.date >= (CURRENT_DATE - (((:'days')::int - 1) * INTERVAL '1 day'))::date
),
ip_rollups AS (
  SELECT i.ip
  FROM api_usage_ip_daily i
  JOIN selected_user u ON u.user_id = i.user_id
  WHERE i.date >= (CURRENT_DATE - (((:'days')::int - 1) * INTERVAL '1 day'))::date
),
route_rollups AS (
  SELECT r.route, r.requests, r.error_requests, r.total_latency_ms
  FROM api_usage_route_daily r
  JOIN selected_user u ON u.user_id = r.user_id
  WHERE r.date >= (CURRENT_DATE - (((:'days')::int - 1) * INTERVAL '1 day'))::date
)
SELECT (SELECT coalesce(sum(total_requests), 0)::int FROM daily_rollups) AS total_requests,
       (SELECT coalesce(sum(success_requests), 0)::int FROM daily_rollups) AS success_requests,
       (SELECT coalesce(sum(error_requests), 0)::int FROM daily_rollups) AS error_requests,
       (SELECT count(DISTINCT nullif(origin, ''))::int FROM origin_rollups) AS unique_origins,
       (SELECT count(DISTINCT ip)::int FROM ip_rollups) AS unique_ips,
       (SELECT count(DISTINCT route)::int FROM route_rollups) AS unique_routes;

ROLLBACK;
