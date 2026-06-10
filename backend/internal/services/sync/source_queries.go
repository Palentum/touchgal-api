package syncsvc

const sourceGamesColumnsSQL = `
SELECT id, unique_id, name, coalesce(introduction, ''), coalesce(banner, ''), coalesce(released, 'unknown'),
       coalesce(content_limit, ''), coalesce(type, '{}'), coalesce(language, '{}'), coalesce(platform, '{}'),
       created, updated, resource_update_time
FROM patch`

const fullSourceGamesPageSQL = sourceGamesColumnsSQL + `
WHERE id > $1
ORDER BY id
LIMIT $2`

const incrementalSourceGamesPageSQL = `
WITH changed AS (
  (SELECT id FROM patch WHERE updated >= $1 AND id > $2 ORDER BY id LIMIT $3)
  UNION
  (SELECT id FROM patch WHERE resource_update_time >= $1 AND id > $2 ORDER BY id LIMIT $3)
)
SELECT p.id, p.unique_id, p.name, coalesce(p.introduction, ''), coalesce(p.banner, ''), coalesce(p.released, 'unknown'),
       coalesce(p.content_limit, ''), coalesce(p.type, '{}'), coalesce(p.language, '{}'), coalesce(p.platform, '{}'),
       p.created, p.updated, p.resource_update_time
FROM patch p
JOIN changed c ON c.id = p.id
ORDER BY p.id
LIMIT $3`

const sourceAliasesByPatchIDsSQL = `
SELECT patch_id, name
FROM patch_alias
WHERE patch_id = ANY($1::int[])
ORDER BY patch_id, name`

const sourceTagsByPatchIDsSQL = `
SELECT r.patch_id, t.name, coalesce(t.alias, '{}'), coalesce(t.source, '')
FROM patch_tag_relation r
JOIN patch_tag t ON t.id = r.tag_id
WHERE r.patch_id = ANY($1::int[])`

const sourceCompaniesByPatchIDsSQL = `
SELECT r.patch_id, c.name, coalesce(c.alias, '{}'), coalesce(c.official_website, '{}'), coalesce(c.primary_language, '{}'), coalesce(c.parent_brand, '{}')
FROM patch_company_relation r
JOIN patch_company c ON c.id = r.company_id
WHERE r.patch_id = ANY($1::int[])`

const sourceRatingsByPatchIDsSQL = `
SELECT patch_id, coalesce(avg_overall, 0), coalesce(count, 0), coalesce(rec_strong_no, 0), coalesce(rec_no, 0),
       coalesce(rec_neutral, 0), coalesce(rec_yes, 0), coalesce(rec_strong_yes, 0),
       coalesce(o1, 0), coalesce(o2, 0), coalesce(o3, 0), coalesce(o4, 0), coalesce(o5, 0),
       coalesce(o6, 0), coalesce(o7, 0), coalesce(o8, 0), coalesce(o9, 0), coalesce(o10, 0)
FROM patch_rating_stat
WHERE patch_id = ANY($1::int[])`
