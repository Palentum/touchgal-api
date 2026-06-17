package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type GameRepo struct{ db Queryer }

func NewGameRepo(db Queryer) *GameRepo { return &GameRepo{db: db} }

const (
	stringListCapHint   = 8
	companyListCapHint  = 4
	resourceListCapHint = 8

	gameSearchSFWPredicate       = "content_limit = 'sfw'"
	gameSearchAllowNsfwPredicate = "content_limit IN ('sfw', 'nsfw')"
	gameDetailSFWPredicate       = "g.content_limit = 'sfw'"
	gameDetailAllowNsfwPredicate = "g.content_limit IN ('sfw', 'nsfw')"

	gameDetailSFWSQL = `
		SELECT g.unique_id, g.name, g.introduction, g.banner_url, g.types, g.platforms, g.languages,
		       g.source_created_at, g.released, g.source_updated_at, g.resource_updated_at,
		       coalesce(r.average_overall, 0), coalesce(r.count, 0), coalesce(r.rec_strong_no, 0), coalesce(r.rec_no, 0), coalesce(r.rec_neutral, 0), coalesce(r.rec_yes, 0), coalesce(r.rec_strong_yes, 0)
		FROM games g
		LEFT JOIN game_rating_stats r ON r.game_unique_id = g.unique_id
		WHERE g.unique_id = $1 AND g.deleted_at IS NULL AND ` + gameDetailSFWPredicate
	gameDetailAllowNsfwSQL = `
		SELECT g.unique_id, g.name, g.introduction, g.banner_url, g.types, g.platforms, g.languages,
		       g.source_created_at, g.released, g.source_updated_at, g.resource_updated_at,
		       coalesce(r.average_overall, 0), coalesce(r.count, 0), coalesce(r.rec_strong_no, 0), coalesce(r.rec_no, 0), coalesce(r.rec_neutral, 0), coalesce(r.rec_yes, 0), coalesce(r.rec_strong_yes, 0)
		FROM games g
		LEFT JOIN game_rating_stats r ON r.game_unique_id = g.unique_id
		WHERE g.unique_id = $1 AND g.deleted_at IS NULL AND ` + gameDetailAllowNsfwPredicate
	gameResourceVisibleSFWSQL = `
		SELECT 1
		FROM games g
		WHERE g.unique_id = $1 AND g.deleted_at IS NULL AND ` + gameDetailSFWPredicate
	gameResourceVisibleAllowNsfwSQL = `
		SELECT 1
		FROM games g
		WHERE g.unique_id = $1 AND g.deleted_at IS NULL AND ` + gameDetailAllowNsfwPredicate
	gameResourceListSFWSQL = `
		SELECT gr.source_resource_id, gr.name, gr.introduction, gr.categories, gr.sizes, gr.published_at
		FROM game_resources gr
		JOIN games g ON g.unique_id = gr.game_unique_id
		WHERE gr.game_unique_id = $1
		  AND gr.resource_type = $2
		  AND g.deleted_at IS NULL
		  AND ` + gameDetailSFWPredicate + `
		ORDER BY gr.published_at DESC, gr.source_resource_id DESC`
	gameResourceListAllowNsfwSQL = `
		SELECT gr.source_resource_id, gr.name, gr.introduction, gr.categories, gr.sizes, gr.published_at
		FROM game_resources gr
		JOIN games g ON g.unique_id = gr.game_unique_id
		WHERE gr.game_unique_id = $1
		  AND gr.resource_type = $2
		  AND g.deleted_at IS NULL
		  AND ` + gameDetailAllowNsfwPredicate + `
		ORDER BY gr.published_at DESC, gr.source_resource_id DESC`

	gameSearchSQLPrefix = `
		WITH ranked_games AS (
			SELECT g.unique_id,
			       g.name,
			       CASE
			         WHEN g.name ILIKE $2 ESCAPE E'\\' THEN 3 + similarity(g.name, $4)
			         WHEN g.name ILIKE $3 ESCAPE E'\\' THEN 2 + similarity(g.name, $4)
			         WHEN g.name ILIKE $1 ESCAPE E'\\' THEN 1 + similarity(g.name, $4)
			         ELSE 0
			       END AS title_rank,
			       CASE
			         WHEN g.name ILIKE $1 ESCAPE E'\\' THEN 0
			         ELSE COALESCE((
			           SELECT max(CASE
			             WHEN a.name ILIKE $2 ESCAPE E'\\' THEN 3 + similarity(a.name, $4)
			             WHEN a.name ILIKE $3 ESCAPE E'\\' THEN 2 + similarity(a.name, $4)
			             WHEN a.name ILIKE $1 ESCAPE E'\\' THEN 1 + similarity(a.name, $4)
			             ELSE 0
			           END)
			           FROM game_aliases a
			           WHERE a.game_unique_id = g.unique_id
			             AND a.name ILIKE $1 ESCAPE E'\\'
			         ), 0)
			       END AS alias_rank,
			       similarity(g.search_text, $4) AS metadata_rank
			FROM games g
			WHERE g.deleted_at IS NULL
			  AND g.`
	gameSearchSQLSuffix = `
			  AND g.search_text ILIKE $1 ESCAPE E'\\'
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
		LIMIT $5 OFFSET $6`
	gameSearchSFWSQL       = gameSearchSQLPrefix + gameSearchSFWPredicate + gameSearchSQLSuffix
	gameSearchAllowNsfwSQL = gameSearchSQLPrefix + gameSearchAllowNsfwPredicate + gameSearchSQLSuffix

	gameSearchCountSQLPrefix = `
		SELECT count(*)
		FROM games
		WHERE deleted_at IS NULL
		  AND `
	gameSearchCountSQLSuffix = `
		  AND search_text ILIKE $1 ESCAPE E'\\'`
	gameSearchCountSFWSQL       = gameSearchCountSQLPrefix + gameSearchSFWPredicate + gameSearchCountSQLSuffix
	gameSearchCountAllowNsfwSQL = gameSearchCountSQLPrefix + gameSearchAllowNsfwPredicate + gameSearchCountSQLSuffix
)

func likePattern(value string, leadingWildcard, trailingWildcard bool) string {
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '%', '_', '\\':
			var b strings.Builder
			b.Grow(len(value)*2 + 2)
			if leadingWildcard {
				b.WriteByte('%')
			}
			b.WriteString(value[:i])
			for ; i < len(value); i++ {
				switch value[i] {
				case '%', '_', '\\':
					b.WriteByte('\\')
				}
				b.WriteByte(value[i])
			}
			if trailingWildcard {
				b.WriteByte('%')
			}
			return b.String()
		}
	}
	if leadingWildcard && trailingWildcard {
		return "%" + value + "%"
	}
	if leadingWildcard {
		return "%" + value
	}
	if trailingWildcard {
		return value + "%"
	}
	return value
}

func likeContainsPattern(value string) string { return likePattern(value, true, true) }
func likeExactPattern(value string) string    { return likePattern(value, false, false) }
func likePrefixPattern(value string) string   { return likePattern(value, false, true) }

func (r *GameRepo) Search(ctx context.Context, keyword string, page, limit int, allowNsfw bool) (model.GameSearchResult, error) {
	offset := (page - 1) * limit
	containsPattern := likeContainsPattern(keyword)
	exactPattern := likeExactPattern(keyword)
	prefixPattern := likePrefixPattern(keyword)
	searchSQL := gameSearchSFWSQL
	countSQL := gameSearchCountSFWSQL
	if allowNsfw {
		searchSQL = gameSearchAllowNsfwSQL
		countSQL = gameSearchCountAllowNsfwSQL
	}
	rows, err := r.db.Query(ctx, searchSQL, containsPattern, exactPattern, prefixPattern, keyword, limit, offset)
	if err != nil {
		return model.GameSearchResult{}, err
	}
	defer rows.Close()

	items := make([]model.GameSearchItem, 0, limit)
	for rows.Next() {
		var item model.GameSearchItem
		if err := rows.Scan(&item.UniqueID, &item.Name); err != nil {
			return model.GameSearchResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return model.GameSearchResult{}, err
	}
	total := 0
	if err := r.db.QueryRow(ctx, countSQL, containsPattern).Scan(&total); err != nil {
		return model.GameSearchResult{}, err
	}
	return model.GameSearchResult{Items: items, Pagination: model.Pagination{Page: page, Limit: limit, Total: total, HasMore: offset+len(items) < total}}, nil
}

func (r *GameRepo) Detail(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (*model.GameDetail, error) {
	detail := &model.GameDetail{}
	var average float64
	var count, recStrongNo, recNo, recNeutral, recYes, recStrongYes int
	detailSQL := gameDetailSFWSQL
	if allowNsfw {
		detailSQL = gameDetailAllowNsfwSQL
	}
	err := r.db.QueryRow(ctx, detailSQL, uniqueID).Scan(&detail.UniqueID, &detail.Name, &detail.Introduction, &detail.BannerURL, &detail.Type, &detail.Platform, &detail.Language, &detail.PublishTime, &detail.ReleaseDate, &detail.UpdatedAt, &detail.ResourceUpdateTime, &average, &count, &recStrongNo, &recNo, &recNeutral, &recYes, &recStrongYes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	detail.Aliases, err = r.stringList(ctx, `SELECT name FROM game_aliases WHERE game_unique_id = $1 ORDER BY name`, uniqueID)
	if err != nil {
		return nil, err
	}
	detail.Tags, err = r.stringList(ctx, `
		SELECT t.name FROM tags t JOIN game_tags gt ON gt.tag_id = t.id WHERE gt.game_unique_id = $1 ORDER BY t.name`, uniqueID)
	if err != nil {
		return nil, err
	}
	detail.Companies, err = r.companies(ctx, uniqueID)
	if err != nil {
		return nil, err
	}
	detail.Rating = model.RatingView{Average: average, Count: count, Recommend: model.RecommendView{StrongNo: recStrongNo, No: recNo, Neutral: recNeutral, Yes: recYes, StrongYes: recStrongYes}}
	detail.TouchGalURL = strings.TrimRight(touchgalSiteURL, "/") + "/" + uniqueID
	return detail, nil
}

func (r *GameRepo) Resources(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (model.GameResourceList, error) {
	return r.resourceList(ctx, uniqueID, touchgalSiteURL, model.ResourceTypeResource, "galgame", allowNsfw)
}

func (r *GameRepo) Patches(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (model.GameResourceList, error) {
	return r.resourceList(ctx, uniqueID, touchgalSiteURL, model.ResourceTypePatch, "patch", allowNsfw)
}

func (r *GameRepo) resourceList(ctx context.Context, uniqueID, touchgalSiteURL, resourceType, resourceSection string, allowNsfw bool) (model.GameResourceList, error) {
	visibleSQL := gameResourceVisibleSFWSQL
	listSQL := gameResourceListSFWSQL
	if allowNsfw {
		visibleSQL = gameResourceVisibleAllowNsfwSQL
		listSQL = gameResourceListAllowNsfwSQL
	}

	var visible int
	err := r.db.QueryRow(ctx, visibleSQL, uniqueID).Scan(&visible)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.GameResourceList{}, model.ErrNotFound
	}
	if err != nil {
		return model.GameResourceList{}, err
	}

	rows, err := r.db.Query(ctx, listSQL, uniqueID, resourceType)
	if err != nil {
		return model.GameResourceList{}, err
	}
	defer rows.Close()

	items := make([]model.GameResourceItem, 0)
	for rows.Next() {
		var sourceResourceID int
		var item model.GameResourceItem
		if err := rows.Scan(&sourceResourceID, &item.Name, &item.Description, &item.Categories, &item.Sizes, &item.PublishTime); err != nil {
			return model.GameResourceList{}, err
		}
		if cap(items) == 0 {
			items = make([]model.GameResourceItem, 0, resourceListCapHint)
		}
		item.DeepLink = touchGalResourceDeepLink(touchgalSiteURL, uniqueID, sourceResourceID, resourceSection)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return model.GameResourceList{}, err
	}
	return model.GameResourceList{Items: items}, nil
}

func touchGalResourceDeepLink(siteURL, uniqueID string, sourceResourceID int, resourceSection string) string {
	return strings.TrimRight(siteURL, "/") + "/" + uniqueID + "?tab=resources&resourceId=" + strconv.Itoa(sourceResourceID) + "&resourceSection=" + resourceSection
}

func (r *GameRepo) stringList(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		if cap(values) == 0 {
			values = make([]string, 0, stringListCapHint)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *GameRepo) companies(ctx context.Context, uniqueID string) ([]model.CompanyView, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.name, c.aliases
		FROM companies c JOIN game_companies gc ON gc.company_id = c.id
		WHERE gc.game_unique_id = $1 ORDER BY c.name`, uniqueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	companies := make([]model.CompanyView, 0)
	for rows.Next() {
		var c model.CompanyView
		if err := rows.Scan(&c.Name, &c.Aliases); err != nil {
			return nil, err
		}
		if cap(companies) == 0 {
			companies = make([]model.CompanyView, 0, companyListCapHint)
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}
