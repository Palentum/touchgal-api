package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type SyncRepo struct{ db Queryer }

func NewSyncRepo(db Queryer) *SyncRepo { return &SyncRepo{db: db} }

func (r *SyncRepo) StartRun(ctx context.Context, mode string) (*model.SyncRun, error) {
	run := &model.SyncRun{ID: uuid.New(), Mode: mode, Status: "running"}
	err := r.db.QueryRow(ctx, `
		INSERT INTO sync_runs (id, mode, status)
		VALUES ($1, $2, 'running')
		RETURNING id, mode, status, started_at, finished_at, source_max_updated_at, games_seen, games_upserted, games_deleted, error_message`,
		run.ID, mode,
	).Scan(&run.ID, &run.Mode, &run.Status, &run.StartedAt, &run.FinishedAt, &run.SourceMaxUpdatedAt, &run.GamesSeen, &run.GamesUpserted, &run.GamesDeleted, &run.ErrorMessage)
	return run, err
}

func (r *SyncRepo) FinishRun(ctx context.Context, id uuid.UUID, status string, sourceMaxUpdatedAt *time.Time, seen, upserted, deleted int, message string) (*model.SyncRun, error) {
	run := &model.SyncRun{}
	err := r.db.QueryRow(ctx, `
		UPDATE sync_runs
		SET status = $2, finished_at = now(), source_max_updated_at = $3, games_seen = $4, games_upserted = $5, games_deleted = $6, error_message = $7
		WHERE id = $1
		RETURNING id, mode, status, started_at, finished_at, source_max_updated_at, games_seen, games_upserted, games_deleted, error_message`,
		id, status, sourceMaxUpdatedAt, seen, upserted, deleted, message,
	).Scan(&run.ID, &run.Mode, &run.Status, &run.StartedAt, &run.FinishedAt, &run.SourceMaxUpdatedAt, &run.GamesSeen, &run.GamesUpserted, &run.GamesDeleted, &run.ErrorMessage)
	return run, err
}

func (r *SyncRepo) LastSuccessMaxUpdatedAt(ctx context.Context) (*time.Time, error) {
	var ts *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT source_max_updated_at FROM sync_runs
		WHERE status = 'success' AND source_max_updated_at IS NOT NULL
		ORDER BY finished_at DESC LIMIT 1`).Scan(&ts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return ts, err
}

func (r *SyncRepo) ListRuns(ctx context.Context, limit int) ([]model.SyncRun, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, mode, status, started_at, finished_at, source_max_updated_at, games_seen, games_upserted, games_deleted, error_message
		FROM sync_runs ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]model.SyncRun, 0)
	for rows.Next() {
		var run model.SyncRun
		if err := rows.Scan(&run.ID, &run.Mode, &run.Status, &run.StartedAt, &run.FinishedAt, &run.SourceMaxUpdatedAt, &run.GamesSeen, &run.GamesUpserted, &run.GamesDeleted, &run.ErrorMessage); err != nil {
			return nil, err
		}
		if cap(runs) == 0 {
			runs = make([]model.SyncRun, 0, positiveCapHint(limit))
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *SyncRepo) UpsertGames(ctx context.Context, games []model.CleanGame) error {
	if len(games) == 0 {
		return nil
	}
	normalized := make([]model.CleanGame, 0, len(games))
	for _, game := range games {
		game.Types = cleanUniqueStrings(game.Types)
		game.Languages = cleanUniqueStrings(game.Languages)
		game.Platforms = cleanUniqueStrings(game.Platforms)
		normalized = append(normalized, game)
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT *
			FROM jsonb_to_recordset($1::jsonb) AS game(
				"UniqueID" text,
				"SourcePatchID" int,
				"Name" text,
				"Introduction" text,
				"BannerURL" text,
				"Released" text,
				"ContentLimit" text,
				"Types" text[],
				"Languages" text[],
				"Platforms" text[],
				"SourceCreatedAt" timestamptz,
				"SourceUpdatedAt" timestamptz,
				"ResourceUpdatedAt" timestamptz
			)
		)
		INSERT INTO games (unique_id, source_patch_id, name, introduction, banner_url, released, content_limit, types, languages, platforms, source_created_at, source_updated_at, resource_updated_at, deleted_at, synced_at)
		SELECT "UniqueID", "SourcePatchID", "Name", "Introduction", "BannerURL", "Released", "ContentLimit", "Types", "Languages", "Platforms", "SourceCreatedAt", "SourceUpdatedAt", "ResourceUpdatedAt", NULL, now()
		FROM payload
		ON CONFLICT (unique_id) DO UPDATE SET
		  source_patch_id = excluded.source_patch_id,
		  name = excluded.name,
		  introduction = excluded.introduction,
		  banner_url = excluded.banner_url,
		  released = excluded.released,
		  content_limit = excluded.content_limit,
		  types = excluded.types,
		  languages = excluded.languages,
		  platforms = excluded.platforms,
		  source_created_at = excluded.source_created_at,
		  source_updated_at = excluded.source_updated_at,
		  resource_updated_at = excluded.resource_updated_at,
		  deleted_at = NULL,
		  synced_at = now()`, string(payload))
	return err
}

func (r *SyncRepo) ReplaceAliasesBatch(ctx context.Context, aliases map[string][]string) error {
	payload := make([]aliasBatchRow, 0, len(aliases))
	for uniqueID, values := range aliases {
		uniqueID = strings.TrimSpace(uniqueID)
		if uniqueID == "" {
			continue
		}
		payload = append(payload, aliasBatchRow{UniqueID: uniqueID, Aliases: cleanUniqueStrings(values)})
	}
	if len(payload) == 0 {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT *
			FROM jsonb_to_recordset($1::jsonb) AS row(unique_id text, aliases text[])
		)
		DELETE FROM game_aliases ga
		USING payload p
		WHERE ga.game_unique_id = p.unique_id`, string(data))
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT *
			FROM jsonb_to_recordset($1::jsonb) AS row(unique_id text, aliases text[])
		)
		INSERT INTO game_aliases (game_unique_id, name)
		SELECT p.unique_id, alias_name.name
		FROM payload p
		CROSS JOIN LATERAL unnest(p.aliases) AS alias_name(name)
		ON CONFLICT DO NOTHING`, string(data))
	return err
}

type aliasBatchRow struct {
	UniqueID string   `json:"unique_id"`
	Aliases  []string `json:"aliases"`
}

func cleanUniqueStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func (r *SyncRepo) ReplaceTagsBatch(ctx context.Context, tags map[string][]model.TagData) error {
	payload := make([]tagBatchRow, 0)
	for uniqueID, values := range tags {
		uniqueID = strings.TrimSpace(uniqueID)
		if uniqueID == "" {
			continue
		}
		cleaned := cleanUniqueTags(values)
		for _, tag := range cleaned {
			if cap(payload) == 0 {
				payload = make([]tagBatchRow, 0, len(tags))
			}
			payload = append(payload, tagBatchRow{UniqueID: uniqueID, Name: tag.Name, Aliases: tag.Aliases, Source: tag.Source})
		}
		if len(cleaned) == 0 {
			if cap(payload) == 0 {
				payload = make([]tagBatchRow, 0, len(tags))
			}
			payload = append(payload, tagBatchRow{UniqueID: uniqueID})
		}
	}
	if len(payload) == 0 {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT DISTINCT unique_id
			FROM jsonb_to_recordset($1::jsonb) AS row(unique_id text, name text, aliases text[], source text)
		)
		DELETE FROM game_tags gt
		USING payload p
		WHERE gt.game_unique_id = p.unique_id`, string(data))
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT *
			FROM jsonb_to_recordset($1::jsonb) AS row(unique_id text, name text, aliases text[], source text)
		), tag_values AS (
			SELECT DISTINCT ON (name) name, aliases, source
			FROM payload
			WHERE name <> ''
			ORDER BY name
		), upserted AS (
			INSERT INTO tags (name, aliases, source)
			SELECT name, aliases, source
			FROM tag_values
			ON CONFLICT (name) DO UPDATE SET aliases = excluded.aliases, source = excluded.source
			RETURNING id, name
		)
		INSERT INTO game_tags (game_unique_id, tag_id)
		SELECT DISTINCT p.unique_id, u.id
		FROM payload p
		JOIN upserted u ON u.name = p.name
		ON CONFLICT DO NOTHING`, string(data))
	return err
}

type tagBatchRow struct {
	UniqueID string   `json:"unique_id"`
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases"`
	Source   string   `json:"source"`
}

func cleanUniqueTags(tags []model.TagData) []model.TagData {
	cleaned := make([]model.TagData, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag.Name = strings.TrimSpace(tag.Name)
		if tag.Name == "" {
			continue
		}
		if _, ok := seen[tag.Name]; ok {
			continue
		}
		tag.Aliases = cleanUniqueStrings(tag.Aliases)
		tag.Source = strings.TrimSpace(tag.Source)
		seen[tag.Name] = struct{}{}
		cleaned = append(cleaned, tag)
	}
	return cleaned
}

func (r *SyncRepo) ReplaceCompaniesBatch(ctx context.Context, companies map[string][]model.CompanyData) error {
	payload := make([]companyBatchRow, 0)
	for uniqueID, values := range companies {
		uniqueID = strings.TrimSpace(uniqueID)
		if uniqueID == "" {
			continue
		}
		cleaned := cleanUniqueCompanies(values)
		for _, company := range cleaned {
			if cap(payload) == 0 {
				payload = make([]companyBatchRow, 0, len(companies))
			}
			payload = append(payload, companyBatchRow{
				UniqueID:         uniqueID,
				Name:             company.Name,
				Aliases:          company.Aliases,
				OfficialWebsites: company.OfficialWebsites,
				PrimaryLanguages: company.PrimaryLanguages,
				ParentBrands:     company.ParentBrands,
			})
		}
		if len(cleaned) == 0 {
			if cap(payload) == 0 {
				payload = make([]companyBatchRow, 0, len(companies))
			}
			payload = append(payload, companyBatchRow{UniqueID: uniqueID})
		}
	}
	if len(payload) == 0 {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT DISTINCT unique_id
			FROM jsonb_to_recordset($1::jsonb) AS row(unique_id text, name text, aliases text[], official_websites text[], primary_languages text[], parent_brands text[])
		)
		DELETE FROM game_companies gc
		USING payload p
		WHERE gc.game_unique_id = p.unique_id`, string(data))
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT *
			FROM jsonb_to_recordset($1::jsonb) AS row(unique_id text, name text, aliases text[], official_websites text[], primary_languages text[], parent_brands text[])
		), company_values AS (
			SELECT DISTINCT ON (name) name, aliases, official_websites, primary_languages, parent_brands
			FROM payload
			WHERE name <> ''
			ORDER BY name
		), upserted AS (
			INSERT INTO companies (name, aliases, official_websites, primary_languages, parent_brands)
			SELECT name, aliases, official_websites, primary_languages, parent_brands
			FROM company_values
			ON CONFLICT (name) DO UPDATE SET aliases = excluded.aliases, official_websites = excluded.official_websites, primary_languages = excluded.primary_languages, parent_brands = excluded.parent_brands
			RETURNING id, name
		)
		INSERT INTO game_companies (game_unique_id, company_id)
		SELECT DISTINCT p.unique_id, u.id
		FROM payload p
		JOIN upserted u ON u.name = p.name
		ON CONFLICT DO NOTHING`, string(data))
	return err
}

type companyBatchRow struct {
	UniqueID         string   `json:"unique_id"`
	Name             string   `json:"name"`
	Aliases          []string `json:"aliases"`
	OfficialWebsites []string `json:"official_websites"`
	PrimaryLanguages []string `json:"primary_languages"`
	ParentBrands     []string `json:"parent_brands"`
}

func cleanUniqueCompanies(companies []model.CompanyData) []model.CompanyData {
	cleaned := make([]model.CompanyData, 0, len(companies))
	seen := make(map[string]struct{}, len(companies))
	for _, company := range companies {
		company.Name = strings.TrimSpace(company.Name)
		if company.Name == "" {
			continue
		}
		if _, ok := seen[company.Name]; ok {
			continue
		}
		company.Aliases = cleanUniqueStrings(company.Aliases)
		company.OfficialWebsites = cleanUniqueStrings(company.OfficialWebsites)
		company.PrimaryLanguages = cleanUniqueStrings(company.PrimaryLanguages)
		company.ParentBrands = cleanUniqueStrings(company.ParentBrands)
		seen[company.Name] = struct{}{}
		cleaned = append(cleaned, company)
	}
	return cleaned
}

func (r *SyncRepo) UpsertRatingsBatch(ctx context.Context, ratings map[string]*model.RatingData, affected []string) error {
	uniqueIDs := cleanUniqueStrings(affected)
	if len(uniqueIDs) == 0 {
		return nil
	}
	payload := make([]ratingBatchRow, 0, len(uniqueIDs))
	for _, uniqueID := range uniqueIDs {
		row := ratingBatchRow{UniqueID: uniqueID}
		if rating := ratings[uniqueID]; rating != nil {
			row.HasRating = true
			row.AverageOverall = rating.AverageOverall
			row.Count = rating.Count
			row.RecStrongNo = rating.RecStrongNo
			row.RecNo = rating.RecNo
			row.RecNeutral = rating.RecNeutral
			row.RecYes = rating.RecYes
			row.RecStrongYes = rating.RecStrongYes
			row.Histogram = &rating.Histogram
		}
		payload = append(payload, row)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		WITH payload AS (
			SELECT *
			FROM jsonb_to_recordset($1::jsonb) AS row(
				unique_id text,
				has_rating bool,
				average_overall double precision,
				count int,
				rec_strong_no int,
				rec_no int,
				rec_neutral int,
				rec_yes int,
				rec_strong_yes int,
				histogram jsonb
			)
		), deleted AS (
			DELETE FROM game_rating_stats grs
			USING payload p
			WHERE grs.game_unique_id = p.unique_id AND NOT p.has_rating
		)
		INSERT INTO game_rating_stats (game_unique_id, average_overall, count, rec_strong_no, rec_no, rec_neutral, rec_yes, rec_strong_yes, histogram, synced_at)
		SELECT unique_id, average_overall, count, rec_strong_no, rec_no, rec_neutral, rec_yes, rec_strong_yes, histogram, now()
		FROM payload
		WHERE has_rating
		ON CONFLICT (game_unique_id) DO UPDATE SET
		  average_overall = excluded.average_overall,
		  count = excluded.count,
		  rec_strong_no = excluded.rec_strong_no,
		  rec_no = excluded.rec_no,
		  rec_neutral = excluded.rec_neutral,
		  rec_yes = excluded.rec_yes,
		  rec_strong_yes = excluded.rec_strong_yes,
		  histogram = excluded.histogram,
		  synced_at = now()`, string(data))
	return err
}

type ratingBatchRow struct {
	UniqueID       string                 `json:"unique_id"`
	HasRating      bool                   `json:"has_rating"`
	AverageOverall float64                `json:"average_overall"`
	Count          int                    `json:"count"`
	RecStrongNo    int                    `json:"rec_strong_no"`
	RecNo          int                    `json:"rec_no"`
	RecNeutral     int                    `json:"rec_neutral"`
	RecYes         int                    `json:"rec_yes"`
	RecStrongYes   int                    `json:"rec_strong_yes"`
	Histogram      *model.RatingHistogram `json:"histogram,omitempty"`
}

func (r *SyncRepo) RefreshSearchTextBatch(ctx context.Context, uniqueIDs []string) error {
	uniqueIDs = cleanUniqueStrings(uniqueIDs)
	if len(uniqueIDs) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		WITH affected AS (
			SELECT unnest($1::text[]) AS unique_id
		), aliases AS (
			SELECT a.game_unique_id, string_agg(a.name, ' ' ORDER BY a.name) AS text
			FROM game_aliases a
			JOIN affected af ON af.unique_id = a.game_unique_id
			GROUP BY a.game_unique_id
		), tag_text AS (
			SELECT gt.game_unique_id, string_agg(t.name, ' ' ORDER BY t.name) AS text
			FROM game_tags gt
			JOIN affected af ON af.unique_id = gt.game_unique_id
			JOIN tags t ON t.id = gt.tag_id
			GROUP BY gt.game_unique_id
		), company_text AS (
			SELECT gc.game_unique_id, string_agg(c.name, ' ' ORDER BY c.name) AS text
			FROM game_companies gc
			JOIN affected af ON af.unique_id = gc.game_unique_id
			JOIN companies c ON c.id = gc.company_id
			GROUP BY gc.game_unique_id
		)
		UPDATE games g SET search_text = concat_ws(' ', g.name, aliases.text, tag_text.text, company_text.text)
		FROM affected af
		LEFT JOIN aliases ON aliases.game_unique_id = af.unique_id
		LEFT JOIN tag_text ON tag_text.game_unique_id = af.unique_id
		LEFT JOIN company_text ON company_text.game_unique_id = af.unique_id
		WHERE g.unique_id = af.unique_id`, uniqueIDs)
	return err
}

func (r *SyncRepo) AddSeenSourcePatchIDs(ctx context.Context, runID uuid.UUID, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO sync_run_seen (run_id, source_patch_id)
		SELECT $1, source_patch_id
		FROM unnest($2::int[]) AS seen(source_patch_id)
		ON CONFLICT DO NOTHING`, runID, cleanUniqueInts(ids))
	return err
}

func cleanUniqueInts(values []int) []int {
	cleaned := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func (r *SyncRepo) MarkDeletedNotSeenByRun(ctx context.Context, runID uuid.UUID) (int, error) {
	cmd, err := r.db.Exec(ctx, `
		UPDATE games g
		SET deleted_at = now()
		WHERE g.deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1
			FROM sync_run_seen s
			WHERE s.run_id = $1 AND s.source_patch_id = g.source_patch_id
		  )`, runID)
	if err != nil {
		return 0, err
	}
	return int(cmd.RowsAffected()), nil
}

func (r *SyncRepo) CleanupSeenSourcePatchIDs(ctx context.Context, runID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sync_run_seen WHERE run_id = $1`, runID)
	return err
}
