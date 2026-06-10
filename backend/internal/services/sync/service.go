package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/repository"
)

const (
	ModeIncremental = "incremental"
	ModeFull        = "full"

	sourceRelationBatchSize = 1000
)

type Service struct {
	cfg    config.Config
	source *pgxpool.Pool
	target *pgxpool.Pool
	repo   *repository.SyncRepo
	log    zerolog.Logger
}

type syncSource struct {
	source SourceGame
	clean  model.CleanGame
}

type sourceRelations struct {
	aliases   map[int][]string
	tags      map[int][]model.TagData
	companies map[int][]model.CompanyData
	ratings   map[int]*model.RatingData
}

func NewService(cfg config.Config, source, target *pgxpool.Pool, repo *repository.SyncRepo, log zerolog.Logger) *Service {
	return &Service{cfg: cfg, source: source, target: target, repo: repo, log: log}
}

func (s *Service) Run(ctx context.Context, mode string) (*model.SyncRun, error) {
	if mode != ModeIncremental && mode != ModeFull {
		return nil, model.ErrInvalidInput
	}
	run, err := s.repo.StartRun(ctx, mode)
	if err != nil {
		return nil, err
	}
	s.log.Debug().Str("mode", mode).Stringer("run_id", run.ID).Msg("sync run started")
	seen, upserted, deleted := 0, 0, 0
	var sourceMax *time.Time
	finish := func(status string, err error) (*model.SyncRun, error) {
		message := ""
		if err != nil {
			message = err.Error()
		}
		updated, finishErr := s.repo.FinishRun(ctx, run.ID, status, sourceMax, seen, upserted, deleted, message)
		if finishErr != nil && err == nil {
			err = finishErr
		}
		event := s.log.Debug().
			Str("mode", mode).
			Stringer("run_id", run.ID).
			Str("status", status).
			Int("seen", seen).
			Int("upserted", upserted).
			Int("deleted", deleted)
		if sourceMax != nil {
			event = event.Time("source_max_updated_at", *sourceMax)
		}
		if finishErr != nil {
			event.Err(finishErr).Msg("sync run finish update failed")
		} else {
			event.Msg("sync run finished")
		}
		return updated, err
	}

	if s.source == nil {
		return finish("failed", errors.New("SOURCE_DATABASE_DSN is not configured"))
	}
	if s.target == nil {
		return finish("failed", errors.New("DATABASE_DSN is not configured"))
	}

	since, err := s.incrementalSince(ctx, mode)
	if err != nil {
		return finish("failed", err)
	}
	window := s.log.Debug().Str("mode", mode).Stringer("run_id", run.ID)
	if since != nil {
		window = window.Time("since", *since)
	}
	window.Msg("sync source query window resolved")
	sources, err := s.querySourceGames(ctx, mode, since)
	if err != nil {
		return finish("failed", err)
	}
	s.log.Debug().Str("mode", mode).Stringer("run_id", run.ID).Int("source_games", len(sources)).Msg("sync source games loaded")

	items := make([]syncSource, 0, len(sources))
	for _, src := range sources {
		clean := MapSourceGame(src, s.cfg.SyncDefaultContentPolicy)
		if clean.UniqueID == "" || clean.Name == "" {
			continue
		}
		items = append(items, syncSource{source: src, clean: clean})
	}

	s.log.Debug().Str("mode", mode).Stringer("run_id", run.ID).Int("syncable_games", len(items)).Msg("sync source games mapped")

	tx, err := s.target.Begin(ctx)
	if err != nil {
		return finish("failed", err)
	}
	defer tx.Rollback(ctx)
	txRepo := repository.NewSyncRepo(tx)
	seenIDs := make([]int, 0, len(items))
	for start := 0; start < len(items); start += sourceRelationBatchSize {
		end := start + sourceRelationBatchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[start:end]
		patchIDs := make([]int, 0, len(batch))
		for _, item := range batch {
			patchIDs = append(patchIDs, item.source.ID)
		}
		relations, err := s.querySourceRelations(ctx, patchIDs)
		if err != nil {
			return finish("failed", err)
		}
		for _, item := range batch {
			src := item.source
			clean := item.clean
			if err := txRepo.UpsertGame(ctx, clean); err != nil {
				return finish("failed", err)
			}
			if err := txRepo.ReplaceAliases(ctx, clean.UniqueID, relations.aliases[src.ID]); err != nil {
				return finish("failed", err)
			}
			if err := txRepo.ReplaceTags(ctx, clean.UniqueID, relations.tags[src.ID]); err != nil {
				return finish("failed", err)
			}
			if err := txRepo.ReplaceCompanies(ctx, clean.UniqueID, relations.companies[src.ID]); err != nil {
				return finish("failed", err)
			}
			if err := txRepo.UpsertRating(ctx, clean.UniqueID, relations.ratings[src.ID]); err != nil {
				return finish("failed", err)
			}
			if err := txRepo.RefreshSearchText(ctx, clean.UniqueID); err != nil {
				return finish("failed", err)
			}
			seen++
			upserted++
			seenIDs = append(seenIDs, src.ID)
			maxUpdated := src.UpdatedAt
			if src.ResourceUpdatedAt.After(maxUpdated) {
				maxUpdated = src.ResourceUpdatedAt
			}
			if sourceMax == nil || maxUpdated.After(*sourceMax) {
				tmp := maxUpdated
				sourceMax = &tmp
			}
		}
	}
	if mode == ModeFull {
		deleted, err = txRepo.MarkDeletedNotSeen(ctx, seenIDs)
		if err != nil {
			return finish("failed", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return finish("failed", err)
	}
	return finish("success", nil)
}

func (s *Service) incrementalSince(ctx context.Context, mode string) (*time.Time, error) {
	if mode == ModeFull {
		return nil, nil
	}
	last, err := s.repo.LastSuccessMaxUpdatedAt(ctx)
	if err != nil || last == nil {
		return last, err
	}
	since := last.Add(-time.Duration(s.cfg.SyncIncrementalSafetyMinutes) * time.Minute)
	return &since, nil
}

func (s *Service) querySourceGames(ctx context.Context, mode string, since *time.Time) ([]SourceGame, error) {
	var rows pgx.Rows
	var err error
	if mode == ModeIncremental && since != nil {
		rows, err = s.source.Query(ctx, incrementalSourceGamesSQL, *since)
	} else {
		rows, err = s.source.Query(ctx, fullSourceGamesSQL)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SourceGame{}
	for rows.Next() {
		var g SourceGame
		if err := rows.Scan(&g.ID, &g.UniqueID, &g.Name, &g.Introduction, &g.Banner, &g.Released, &g.ContentLimit, &g.Types, &g.Languages, &g.Platforms, &g.CreatedAt, &g.UpdatedAt, &g.ResourceUpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (s *Service) querySourceRelations(ctx context.Context, patchIDs []int) (sourceRelations, error) {
	relations := sourceRelations{
		aliases:   make(map[int][]string, len(patchIDs)),
		tags:      make(map[int][]model.TagData, len(patchIDs)),
		companies: make(map[int][]model.CompanyData, len(patchIDs)),
		ratings:   make(map[int]*model.RatingData, len(patchIDs)),
	}
	if len(patchIDs) == 0 {
		return relations, nil
	}

	rows, err := s.source.Query(ctx, sourceAliasesByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var value string
		if err := rows.Scan(&patchID, &value); err != nil {
			rows.Close()
			return relations, err
		}
		relations.aliases[patchID] = append(relations.aliases[patchID], value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return relations, err
	}
	rows.Close()

	rows, err = s.source.Query(ctx, sourceTagsByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var item model.TagData
		if err := rows.Scan(&patchID, &item.Name, &item.Aliases, &item.Source); err != nil {
			rows.Close()
			return relations, err
		}
		relations.tags[patchID] = append(relations.tags[patchID], item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return relations, err
	}
	rows.Close()

	rows, err = s.source.Query(ctx, sourceCompaniesByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var item model.CompanyData
		if err := rows.Scan(&patchID, &item.Name, &item.Aliases, &item.OfficialWebsites, &item.PrimaryLanguages, &item.ParentBrands); err != nil {
			rows.Close()
			return relations, err
		}
		relations.companies[patchID] = append(relations.companies[patchID], item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return relations, err
	}
	rows.Close()

	rows, err = s.source.Query(ctx, sourceRatingsByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var r model.RatingData
		var o1, o2, o3, o4, o5, o6, o7, o8, o9, o10 int
		if err := rows.Scan(&patchID, &r.AverageOverall, &r.Count, &r.RecStrongNo, &r.RecNo, &r.RecNeutral, &r.RecYes, &r.RecStrongYes, &o1, &o2, &o3, &o4, &o5, &o6, &o7, &o8, &o9, &o10); err != nil {
			rows.Close()
			return relations, err
		}
		r.Histogram = map[string]int{"1": o1, "2": o2, "3": o3, "4": o4, "5": o5, "6": o6, "7": o7, "8": o8, "9": o9, "10": o10}
		relations.ratings[patchID] = &r
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return relations, err
	}
	rows.Close()

	return relations, nil
}

func (s *Service) EnsureSourceReadOnly(ctx context.Context) error {
	if s.source == nil {
		return errors.New("source database is not configured")
	}
	_, err := s.source.Exec(ctx, `SET TRANSACTION READ ONLY`)
	if err != nil {
		s.log.Warn().Err(err).Msg("source read-only transaction check failed")
	}
	return nil
}

func (s *Service) String() string {
	return fmt.Sprintf("sync interval=%dm full=%dh", s.cfg.SyncIntervalMinutes, s.cfg.SyncFullIntervalHours)
}
