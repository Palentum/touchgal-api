package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type GameRepo struct{ db Queryer }

func NewGameRepo(db Queryer) *GameRepo { return &GameRepo{db: db} }

func likeContainsPattern(value string) string {
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '%', '_', '\\':
			var b strings.Builder
			b.Grow(len(value) + 4)
			b.WriteByte('%')
			b.WriteString(value[:i])
			for ; i < len(value); i++ {
				switch value[i] {
				case '%', '_', '\\':
					b.WriteByte('\\')
				}
				b.WriteByte(value[i])
			}
			b.WriteByte('%')
			return b.String()
		}
	}
	return "%" + value + "%"
}

func (r *GameRepo) Search(ctx context.Context, keyword string, page, limit int) (model.GameSearchResult, error) {
	offset := (page - 1) * limit
	pattern := likeContainsPattern(keyword)
	rows, err := r.db.Query(ctx, `
		SELECT unique_id, name
		FROM games
		WHERE deleted_at IS NULL
		  AND content_limit = 'sfw'
		  AND search_text ILIKE $1 ESCAPE E'\\'
		ORDER BY name ASC, unique_id ASC
		LIMIT $2 OFFSET $3`, pattern, limit, offset)
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
	if err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM games
		WHERE deleted_at IS NULL
		  AND content_limit = 'sfw'
		  AND search_text ILIKE $1 ESCAPE E'\\'`, pattern).Scan(&total); err != nil {
		return model.GameSearchResult{}, err
	}
	return model.GameSearchResult{Items: items, Pagination: model.Pagination{Page: page, Limit: limit, Total: total, HasMore: offset+len(items) < total}}, nil
}

func (r *GameRepo) Detail(ctx context.Context, uniqueID, touchgalSiteURL string) (*model.GameDetail, error) {
	detail := &model.GameDetail{}
	var average float64
	var count, recStrongNo, recNo, recNeutral, recYes, recStrongYes int
	err := r.db.QueryRow(ctx, `
		SELECT g.unique_id, g.name, g.introduction, g.banner_url, g.types, g.platforms, g.languages,
		       g.source_created_at, g.released, g.source_updated_at, g.resource_updated_at,
		       coalesce(r.average_overall, 0), coalesce(r.count, 0), coalesce(r.rec_strong_no, 0), coalesce(r.rec_no, 0), coalesce(r.rec_neutral, 0), coalesce(r.rec_yes, 0), coalesce(r.rec_strong_yes, 0)
		FROM games g
		LEFT JOIN game_rating_stats r ON r.game_unique_id = g.unique_id
		WHERE g.unique_id = $1 AND g.deleted_at IS NULL AND g.content_limit = 'sfw'`, uniqueID,
	).Scan(&detail.UniqueID, &detail.Name, &detail.Introduction, &detail.BannerURL, &detail.Type, &detail.Platform, &detail.Language, &detail.PublishTime, &detail.ReleaseDate, &detail.UpdatedAt, &detail.ResourceUpdateTime, &average, &count, &recStrongNo, &recNo, &recNeutral, &recYes, &recStrongYes)
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

func (r *GameRepo) stringList(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
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
	companies := []model.CompanyView{}
	for rows.Next() {
		var c model.CompanyView
		if err := rows.Scan(&c.Name, &c.Aliases); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}
