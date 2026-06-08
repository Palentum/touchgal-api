package syncsvc

const fullSourceGamesSQL = `
SELECT id, unique_id, name, coalesce(introduction, ''), coalesce(banner, ''), coalesce(released, 'unknown'),
       coalesce(content_limit, ''), coalesce(type, '{}'), coalesce(language, '{}'), coalesce(platform, '{}'),
       created, updated, resource_update_time
FROM patch`

const incrementalSourceGamesSQL = fullSourceGamesSQL + `
WHERE updated >= $1 OR resource_update_time >= $1`

const sourceAliasesSQL = `
SELECT name FROM patch_alias WHERE patch_id = $1 ORDER BY name`

const sourceTagsSQL = `
SELECT t.name, coalesce(t.alias, '{}'), coalesce(t.source, '')
FROM patch_tag_relation r
JOIN patch_tag t ON t.id = r.tag_id
WHERE r.patch_id = $1`

const sourceCompaniesSQL = `
SELECT c.name, coalesce(c.alias, '{}'), coalesce(c.official_website, '{}'), coalesce(c.primary_language, '{}'), coalesce(c.parent_brand, '{}')
FROM patch_company_relation r
JOIN patch_company c ON c.id = r.company_id
WHERE r.patch_id = $1`

const sourceRatingSQL = `
SELECT coalesce(avg_overall, 0), coalesce(count, 0), coalesce(rec_strong_no, 0), coalesce(rec_no, 0),
       coalesce(rec_neutral, 0), coalesce(rec_yes, 0), coalesce(rec_strong_yes, 0),
       coalesce(o1, 0), coalesce(o2, 0), coalesce(o3, 0), coalesce(o4, 0), coalesce(o5, 0),
       coalesce(o6, 0), coalesce(o7, 0), coalesce(o8, 0), coalesce(o9, 0), coalesce(o10, 0)
FROM patch_rating_stat WHERE patch_id = $1`
