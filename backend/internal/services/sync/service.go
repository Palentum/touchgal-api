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
)

type Service struct {
	cfg    config.Config
	source *pgxpool.Pool
	target *pgxpool.Pool
	repo   *repository.SyncRepo
	log    zerolog.Logger
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
	sources, err := s.querySourceGames(ctx, mode, since)
	if err != nil {
		return finish("failed", err)
	}

	tx, err := s.target.Begin(ctx)
	if err != nil {
		return finish("failed", err)
	}
	defer tx.Rollback(ctx)
	txRepo := repository.NewSyncRepo(tx)
	seenIDs := make([]int, 0, len(sources))
	for _, src := range sources {
		clean := MapSourceGame(src, s.cfg.SyncDefaultContentPolicy)
		if clean.UniqueID == "" || clean.Name == "" {
			continue
		}
		if err := txRepo.UpsertGame(ctx, clean); err != nil {
			return finish("failed", err)
		}
		aliases, err := s.queryAliases(ctx, src.ID)
		if err != nil {
			return finish("failed", err)
		}
		if err := txRepo.ReplaceAliases(ctx, clean.UniqueID, aliases); err != nil {
			return finish("failed", err)
		}
		tags, err := s.queryTags(ctx, src.ID)
		if err != nil {
			return finish("failed", err)
		}
		if err := txRepo.ReplaceTags(ctx, clean.UniqueID, tags); err != nil {
			return finish("failed", err)
		}
		companies, err := s.queryCompanies(ctx, src.ID)
		if err != nil {
			return finish("failed", err)
		}
		if err := txRepo.ReplaceCompanies(ctx, clean.UniqueID, companies); err != nil {
			return finish("failed", err)
		}
		rating, err := s.queryRating(ctx, src.ID)
		if err != nil {
			return finish("failed", err)
		}
		if err := txRepo.UpsertRating(ctx, clean.UniqueID, rating); err != nil {
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

func (s *Service) queryAliases(ctx context.Context, patchID int) ([]string, error) {
	rows, err := s.source.Query(ctx, sourceAliasesSQL, patchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func (s *Service) queryTags(ctx context.Context, patchID int) ([]model.TagData, error) {
	rows, err := s.source.Query(ctx, sourceTagsSQL, patchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.TagData{}
	for rows.Next() {
		var item model.TagData
		if err := rows.Scan(&item.Name, &item.Aliases, &item.Source); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryCompanies(ctx context.Context, patchID int) ([]model.CompanyData, error) {
	rows, err := s.source.Query(ctx, sourceCompaniesSQL, patchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.CompanyData{}
	for rows.Next() {
		var item model.CompanyData
		if err := rows.Scan(&item.Name, &item.Aliases, &item.OfficialWebsites, &item.PrimaryLanguages, &item.ParentBrands); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryRating(ctx context.Context, patchID int) (*model.RatingData, error) {
	var r model.RatingData
	var o1, o2, o3, o4, o5, o6, o7, o8, o9, o10 int
	err := s.source.QueryRow(ctx, sourceRatingSQL, patchID).Scan(&r.AverageOverall, &r.Count, &r.RecStrongNo, &r.RecNo, &r.RecNeutral, &r.RecYes, &r.RecStrongYes, &o1, &o2, &o3, &o4, &o5, &o6, &o7, &o8, &o9, &o10)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Histogram = map[string]int{"1": o1, "2": o2, "3": o3, "4": o4, "5": o5, "6": o6, "7": o7, "8": o8, "9": o9, "10": o10}
	return &r, nil
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
