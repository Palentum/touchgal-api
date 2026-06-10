\if :{?source_last_id}
\else
\set source_last_id 0
\endif
\if :{?source_limit}
\else
\set source_limit 1000
\endif
\if :{?source_window}
\else
\set source_window '1 day'
\endif

BEGIN READ ONLY;
SET LOCAL statement_timeout = '30s';

\echo 'source-db full sync keyset page query'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, unique_id, name, coalesce(introduction, ''), coalesce(banner, ''), coalesce(released, 'unknown'),
       coalesce(content_limit, ''), coalesce(type, '{}'), coalesce(language, '{}'), coalesce(platform, '{}'),
       created, updated, resource_update_time
FROM patch
WHERE id > (:'source_last_id')::int
ORDER BY id
LIMIT (:'source_limit')::int;

\echo 'source-db incremental sync changed page query'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
WITH changed AS (
  (SELECT id FROM patch WHERE updated >= (now() - (:'source_window')::interval) AND id > (:'source_last_id')::int ORDER BY id LIMIT (:'source_limit')::int)
  UNION
  (SELECT id FROM patch WHERE resource_update_time >= (now() - (:'source_window')::interval) AND id > (:'source_last_id')::int ORDER BY id LIMIT (:'source_limit')::int)
)
SELECT p.id, p.unique_id, p.name, coalesce(p.introduction, ''), coalesce(p.banner, ''), coalesce(p.released, 'unknown'),
       coalesce(p.content_limit, ''), coalesce(p.type, '{}'), coalesce(p.language, '{}'), coalesce(p.platform, '{}'),
       p.created, p.updated, p.resource_update_time
FROM patch p
JOIN changed c ON c.id = p.id
ORDER BY p.id
LIMIT (:'source_limit')::int;

ROLLBACK;
