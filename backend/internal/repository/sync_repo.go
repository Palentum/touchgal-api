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
	runs := []model.SyncRun{}
	for rows.Next() {
		var run model.SyncRun
		if err := rows.Scan(&run.ID, &run.Mode, &run.Status, &run.StartedAt, &run.FinishedAt, &run.SourceMaxUpdatedAt, &run.GamesSeen, &run.GamesUpserted, &run.GamesDeleted, &run.ErrorMessage); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *SyncRepo) UpsertGame(ctx context.Context, game model.CleanGame) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO games (unique_id, source_patch_id, name, introduction, banner_url, released, content_limit, types, languages, platforms, source_created_at, source_updated_at, resource_updated_at, deleted_at, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULL, now())
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
		  synced_at = now()`,
		game.UniqueID, game.SourcePatchID, game.Name, game.Introduction, game.BannerURL, game.Released, game.ContentLimit, game.Types, game.Languages, game.Platforms, game.SourceCreatedAt, game.SourceUpdatedAt, game.ResourceUpdatedAt)
	return err
}

func (r *SyncRepo) ReplaceAliases(ctx context.Context, uniqueID string, aliases []string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM game_aliases WHERE game_unique_id = $1`, uniqueID); err != nil {
		return err
	}
	cleaned := cleanUniqueStrings(aliases)
	if len(cleaned) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO game_aliases (game_unique_id, name)
		SELECT $1, alias_name.name
		FROM unnest($2::text[]) AS alias_name(name)
		ON CONFLICT DO NOTHING`, uniqueID, cleaned)
	return err
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

func (r *SyncRepo) ReplaceTags(ctx context.Context, uniqueID string, tags []model.TagData) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM game_tags WHERE game_unique_id = $1`, uniqueID); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		var id int64
		err := r.db.QueryRow(ctx, `
			INSERT INTO tags (name, aliases, source) VALUES ($1, $2, $3)
			ON CONFLICT (name) DO UPDATE SET aliases = excluded.aliases, source = excluded.source
			RETURNING id`, name, tag.Aliases, tag.Source).Scan(&id)
		if err != nil {
			return err
		}
		if _, err := r.db.Exec(ctx, `INSERT INTO game_tags (game_unique_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, uniqueID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *SyncRepo) ReplaceCompanies(ctx context.Context, uniqueID string, companies []model.CompanyData) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM game_companies WHERE game_unique_id = $1`, uniqueID); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(companies))
	for _, company := range companies {
		name := strings.TrimSpace(company.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		var id int64
		err := r.db.QueryRow(ctx, `
			INSERT INTO companies (name, aliases, official_websites, primary_languages, parent_brands) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (name) DO UPDATE SET aliases = excluded.aliases, official_websites = excluded.official_websites, primary_languages = excluded.primary_languages, parent_brands = excluded.parent_brands
			RETURNING id`, name, company.Aliases, company.OfficialWebsites, company.PrimaryLanguages, company.ParentBrands).Scan(&id)
		if err != nil {
			return err
		}
		if _, err := r.db.Exec(ctx, `INSERT INTO game_companies (game_unique_id, company_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, uniqueID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *SyncRepo) UpsertRating(ctx context.Context, uniqueID string, rating *model.RatingData) error {
	if rating == nil {
		_, err := r.db.Exec(ctx, `DELETE FROM game_rating_stats WHERE game_unique_id = $1`, uniqueID)
		return err
	}
	histogram, err := json.Marshal(rating.Histogram)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO game_rating_stats (game_unique_id, average_overall, count, rec_strong_no, rec_no, rec_neutral, rec_yes, rec_strong_yes, histogram, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, now())
		ON CONFLICT (game_unique_id) DO UPDATE SET
		  average_overall = excluded.average_overall,
		  count = excluded.count,
		  rec_strong_no = excluded.rec_strong_no,
		  rec_no = excluded.rec_no,
		  rec_neutral = excluded.rec_neutral,
		  rec_yes = excluded.rec_yes,
		  rec_strong_yes = excluded.rec_strong_yes,
		  histogram = excluded.histogram,
		  synced_at = now()`,
		uniqueID, rating.AverageOverall, rating.Count, rating.RecStrongNo, rating.RecNo, rating.RecNeutral, rating.RecYes, rating.RecStrongYes, string(histogram))
	return err
}

func (r *SyncRepo) RefreshSearchText(ctx context.Context, uniqueID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE games g SET search_text = concat_ws(' ',
		  g.name,
		  (SELECT string_agg(a.name, ' ') FROM game_aliases a WHERE a.game_unique_id = g.unique_id),
		  (SELECT string_agg(t.name, ' ') FROM tags t JOIN game_tags gt ON gt.tag_id = t.id WHERE gt.game_unique_id = g.unique_id),
		  (SELECT string_agg(c.name, ' ') FROM companies c JOIN game_companies gc ON gc.company_id = c.id WHERE gc.game_unique_id = g.unique_id)
		)
		WHERE g.unique_id = $1`, uniqueID)
	return err
}

func (r *SyncRepo) MarkDeletedNotSeen(ctx context.Context, seenSourcePatchIDs []int) (int, error) {
	if len(seenSourcePatchIDs) == 0 {
		cmd, err := r.db.Exec(ctx, `UPDATE games SET deleted_at = now() WHERE deleted_at IS NULL`)
		if err != nil {
			return 0, err
		}
		return int(cmd.RowsAffected()), nil
	}
	cmd, err := r.db.Exec(ctx, `UPDATE games SET deleted_at = now() WHERE deleted_at IS NULL AND NOT (source_patch_id = ANY($1::int[]))`, seenSourcePatchIDs)
	if err != nil {
		return 0, err
	}
	return int(cmd.RowsAffected()), nil
}
