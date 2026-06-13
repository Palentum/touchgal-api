package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
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

	sourceGamePageSize         = 1000
	sourceReadOnlyCheckTimeout = 10 * time.Second
)

var sourceReadOnlyTables = [...]string{
	"patch",
	"patch_alias",
	"patch_tag_relation",
	"patch_tag",
	"patch_company_relation",
	"patch_company",
	"patch_rating_stat",
}

const sourceReadOnlyStatusSQL = `
SELECT current_setting('transaction_read_only'),
       has_database_privilege(current_database(), 'CREATE'),
       has_database_privilege(current_database(), 'TEMPORARY'),
       current_setting('server_version_num')::integer >= 170000`

const sourceWritePrivilegeCheckSQL = `
WITH required_tables(name) AS (
	SELECT unnest($1::text[])
), resolved_tables AS (
	SELECT required_tables.name,
	       c.oid,
	       c.oid::regclass::text AS relation,
	       c.relnamespace
	FROM required_tables
	LEFT JOIN pg_class c ON c.oid = to_regclass(required_tables.name)
), checked_tables AS (
	SELECT name,
	       coalesce(relation, '') AS relation,
	       oid IS NULL AS missing,
	       CASE WHEN oid IS NULL THEN false ELSE has_table_privilege(oid, 'SELECT') END AS can_select,
	       CASE WHEN oid IS NULL THEN false ELSE (
		       has_table_privilege(oid, 'INSERT') OR
		       has_table_privilege(oid, 'UPDATE') OR
		       has_table_privilege(oid, 'DELETE') OR
		       has_table_privilege(oid, 'TRUNCATE') OR
		       has_table_privilege(oid, 'REFERENCES') OR
		       has_table_privilege(oid, 'TRIGGER') OR
		       CASE WHEN $2::boolean THEN has_table_privilege(oid, 'MAINTAIN') ELSE false END
	       ) END AS table_writable,
	       CASE WHEN relnamespace IS NULL THEN false ELSE has_schema_privilege(relnamespace, 'CREATE') END AS schema_writable
	FROM resolved_tables
)
SELECT name, relation, missing, can_select, table_writable, schema_writable
FROM checked_tables
WHERE missing OR NOT can_select OR table_writable OR schema_writable
ORDER BY name
LIMIT 1`

type runStore interface {
	StartRun(ctx context.Context, mode string) (*model.SyncRun, error)
	FinishRun(ctx context.Context, id uuid.UUID, status string, sourceMaxUpdatedAt *time.Time, seen, upserted, deleted int, message string) (*model.SyncRun, error)
	LastSuccessMaxUpdatedAt(ctx context.Context) (*time.Time, error)
}
type syncRunLockFunc func(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) (*repository.SyncRunLock, bool, error)

type sourceReadOnlyQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Service struct {
	cfg    config.Config
	source *pgxpool.Pool
	target *pgxpool.Pool
	repo   runStore
	log    zerolog.Logger

	acquireLock syncRunLockFunc

	checkSourceReadOnly func(context.Context) error

	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
	wg              sync.WaitGroup

	runMu      sync.Mutex
	runRunning bool
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
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	return &Service{cfg: cfg, source: source, target: target, repo: repo, log: log, lifecycleCtx: lifecycleCtx, lifecycleCancel: lifecycleCancel, acquireLock: repository.TryAcquireSyncRunLock}
}

func (s *Service) Stop() {
	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}
	s.wg.Wait()
}

func (s *Service) backgroundContext() context.Context {
	if s.lifecycleCtx != nil {
		return s.lifecycleCtx
	}
	return context.Background()
}

func validateMode(mode string) error {
	if mode != ModeIncremental && mode != ModeFull {
		return model.ErrInvalidInput
	}
	return nil
}

func (s *Service) acquireRun() bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.runRunning {
		return false
	}
	s.runRunning = true
	return true
}

func (s *Service) releaseRun() {
	s.runMu.Lock()
	s.runRunning = false
	s.runMu.Unlock()
}

func (s *Service) ensureSourceReadOnly(ctx context.Context) error {
	if s.checkSourceReadOnly != nil {
		return s.checkSourceReadOnly(ctx)
	}
	return s.EnsureSourceReadOnly(ctx)
}

func (s *Service) Start(_ context.Context, mode string) (*model.SyncRun, error) {
	if err := validateMode(mode); err != nil {
		return nil, err
	}
	if !s.acquireRun() {
		return nil, model.ErrSyncRunning
	}
	s.wg.Add(1)
	runCtx := s.backgroundContext()
	lock, runRepo, err := s.acquireDistributedRunLock(runCtx)
	if err != nil {
		s.wg.Done()
		s.releaseRun()
		return nil, err
	}
	if err := s.ensureSourceReadOnly(runCtx); err != nil {
		s.releaseSyncRunLock(runCtx, lock)
		s.wg.Done()
		s.releaseRun()
		return nil, err
	}
	run, err := s.startRun(runCtx, mode, runRepo)
	if err != nil {
		s.releaseSyncRunLock(runCtx, lock)
		s.wg.Done()
		s.releaseRun()
		return nil, err
	}
	go func() {
		defer s.wg.Done()
		defer s.releaseRun()
		if _, err := s.runStarted(runCtx, run, lock, runRepo); err != nil {
			s.log.Error().Err(err).Str("mode", run.Mode).Stringer("run_id", run.ID).Msg("sync failed")
		}
	}()
	return run, nil
}

func (s *Service) Run(ctx context.Context, mode string) (*model.SyncRun, error) {
	if err := validateMode(mode); err != nil {
		return nil, err
	}
	if !s.acquireRun() {
		return nil, model.ErrSyncRunning
	}
	s.wg.Add(1)
	defer s.wg.Done()
	defer s.releaseRun()
	lock, runRepo, err := s.acquireDistributedRunLock(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSourceReadOnly(ctx); err != nil {
		s.releaseSyncRunLock(ctx, lock)
		return nil, err
	}
	run, err := s.startRun(ctx, mode, runRepo)
	if err != nil {
		s.releaseSyncRunLock(ctx, lock)
		return nil, err
	}
	return s.runStarted(ctx, run, lock, runRepo)
}

func (s *Service) startRun(ctx context.Context, mode string, repo runStore) (*model.SyncRun, error) {
	if repo == nil {
		repo = s.repo
	}
	run, err := repo.StartRun(ctx, mode)
	if err != nil {
		return nil, err
	}
	s.log.Debug().Str("mode", mode).Stringer("run_id", run.ID).Msg("sync run started")
	return run, nil
}

func sourceReadOnlyCheckContext(ctx context.Context, cfg config.PostgresConfig) (context.Context, context.CancelFunc) {
	timeout := sourceReadOnlyCheckTimeout
	if cfg.QueryTimeout > 0 && cfg.QueryTimeout < timeout {
		timeout = cfg.QueryTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *Service) acquireDistributedRunLock(ctx context.Context) (*repository.SyncRunLock, runStore, error) {
	if s.target == nil {
		return nil, s.repo, nil
	}
	acquire := s.acquireLock
	if acquire == nil {
		acquire = repository.TryAcquireSyncRunLock
	}
	lock, locked, err := acquire(ctx, s.target, s.cfg.SyncDatabasePool.QueryTimeout)
	if err != nil {
		return nil, nil, err
	}
	if !locked {
		return nil, nil, model.ErrSyncRunning
	}
	if lock == nil {
		return nil, nil, errors.New("sync run lock was not returned")
	}
	return lock, lock.Repo(), nil
}

func (s *Service) releaseSyncRunLock(ctx context.Context, lock *repository.SyncRunLock) {
	if err := s.releaseSyncRunLockError(ctx, lock); err != nil {
		s.log.Warn().Err(err).Msg("sync run lock release failed")
	}
}

func (s *Service) releaseSyncRunLockError(ctx context.Context, lock *repository.SyncRunLock) error {
	if lock == nil {
		return nil
	}
	releaseCtx, cancel := s.finishContext(ctx)
	defer cancel()
	return lock.Release(releaseCtx)
}

func (s *Service) finishContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), s.cfg.EffectiveSyncRunFinishTimeout())
}

func optionalTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *Service) runStarted(ctx context.Context, run *model.SyncRun, lock *repository.SyncRunLock, runRepo runStore) (*model.SyncRun, error) {
	mode := run.Mode
	seen, upserted, deleted := 0, 0, 0
	var sourceMax *time.Time
	if runRepo == nil {
		runRepo = s.repo
	}
	finishRepo := runRepo
	finish := func(status string, err error) (*model.SyncRun, error) {
		message := ""
		if err != nil {
			message = err.Error()
		}
		finishCtx, cancel := s.finishContext(ctx)
		defer cancel()
		updated, finishErr := finishRepo.FinishRun(finishCtx, run.ID, status, sourceMax, seen, upserted, deleted, message)
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
	defer s.releaseSyncRunLock(ctx, lock)

	if s.source == nil {
		return finish("failed", errors.New("SOURCE_DATABASE_DSN is not configured"))
	}
	if s.target == nil {
		return finish("failed", errors.New("DATABASE_DSN is not configured"))
	}

	if lock == nil {
		return finish("failed", errors.New("sync run lock is not acquired"))
	}
	if mode == ModeFull {
		defer s.cleanupSeenSourcePatchIDs(ctx, lock, run.ID)
	}

	since, err := s.incrementalSince(ctx, mode, runRepo)
	if err != nil {
		return finish("failed", err)
	}
	window := s.log.Debug().Str("mode", mode).Stringer("run_id", run.ID)
	if since != nil {
		window = window.Time("since", *since)
	}
	window.Msg("sync source query window resolved")

	afterID := 0
	page := 0
	for {
		sources, err := s.querySourceGamePage(ctx, mode, since, afterID, sourceGamePageSize)
		if err != nil {
			return finish("failed", err)
		}
		if len(sources) == 0 {
			break
		}
		afterID = sources[len(sources)-1].ID
		page++
		items := make([]syncSource, 0, len(sources))
		patchIDs := make([]int, 0, len(sources))
		for _, src := range sources {
			clean := MapSourceGame(src, s.cfg.SyncDefaultContentPolicy)
			if clean.UniqueID == "" || clean.Name == "" {
				continue
			}
			items = append(items, syncSource{source: src, clean: clean})
			patchIDs = append(patchIDs, src.ID)
		}
		if len(items) == 0 {
			s.log.Debug().
				Str("mode", mode).
				Stringer("run_id", run.ID).
				Int("page", page).
				Int("source_games", len(sources)).
				Msg("sync source page contained no syncable games")
			continue
		}

		relations, err := s.querySourceRelations(ctx, patchIDs)
		if err != nil {
			return finish("failed", err)
		}
		batchSeen, batchUpserted, batchSourceMax, err := s.writeTargetBatch(ctx, lock, run.ID, mode, items, relations)
		if err != nil {
			return finish("failed", err)
		}
		seen += batchSeen
		upserted += batchUpserted
		if batchSourceMax != nil && (sourceMax == nil || batchSourceMax.After(*sourceMax)) {
			tmp := *batchSourceMax
			sourceMax = &tmp
		}
		s.log.Debug().
			Str("mode", mode).
			Stringer("run_id", run.ID).
			Int("page", page).
			Int("source_games", len(sources)).
			Int("syncable_games", len(items)).
			Msg("sync source page committed")
	}

	if mode == ModeFull {
		deleted, err = s.markDeletedNotSeen(ctx, lock, run.ID)
		if err != nil {
			return finish("failed", err)
		}
	}

	return finish("success", nil)
}

func (s *Service) writeTargetBatch(ctx context.Context, lock *repository.SyncRunLock, runID uuid.UUID, mode string, items []syncSource, relations sourceRelations) (int, int, *time.Time, error) {
	if len(items) == 0 {
		return 0, 0, nil, nil
	}
	games := make([]model.CleanGame, 0, len(items))
	uniqueIDs := make([]string, 0, len(items))
	patchIDs := make([]int, 0, len(items))
	aliases := make(map[string][]string, len(items))
	tags := make(map[string][]model.TagData, len(items))
	companies := make(map[string][]model.CompanyData, len(items))
	ratings := make(map[string]*model.RatingData, len(items))
	var sourceMax *time.Time
	for _, item := range items {
		src := item.source
		clean := item.clean
		uniqueID := clean.UniqueID
		games = append(games, clean)
		uniqueIDs = append(uniqueIDs, uniqueID)
		patchIDs = append(patchIDs, src.ID)
		aliases[uniqueID] = relations.aliases[src.ID]
		tags[uniqueID] = relations.tags[src.ID]
		companies[uniqueID] = relations.companies[src.ID]
		ratings[uniqueID] = relations.ratings[src.ID]
		maxUpdated := src.UpdatedAt
		if src.ResourceUpdatedAt.After(maxUpdated) {
			maxUpdated = src.ResourceUpdatedAt
		}
		if sourceMax == nil || maxUpdated.After(*sourceMax) {
			tmp := maxUpdated
			sourceMax = &tmp
		}
	}

	beginCtx, cancel := optionalTimeoutContext(ctx, s.cfg.SyncDatabasePool.QueryTimeout)
	tx, err := lock.Begin(beginCtx)
	cancel()
	if err != nil {
		return 0, 0, nil, err
	}
	defer func() {
		rollbackCtx, cancel := s.finishContext(ctx)
		defer cancel()
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.log.Warn().Err(err).Stringer("run_id", runID).Msg("sync target batch rollback failed")
		}
	}()
	txRepo := repository.NewSyncRepo(repository.WithQueryTimeout(tx, s.cfg.SyncDatabasePool.QueryTimeout))
	if err := txRepo.UpsertGames(ctx, games); err != nil {
		return 0, 0, nil, err
	}
	if err := txRepo.ReplaceAliasesBatch(ctx, aliases); err != nil {
		return 0, 0, nil, err
	}
	if err := txRepo.ReplaceTagsBatch(ctx, tags); err != nil {
		return 0, 0, nil, err
	}
	if err := txRepo.ReplaceCompaniesBatch(ctx, companies); err != nil {
		return 0, 0, nil, err
	}
	if err := txRepo.UpsertRatingsBatch(ctx, ratings, uniqueIDs); err != nil {
		return 0, 0, nil, err
	}
	if err := txRepo.RefreshSearchTextBatch(ctx, uniqueIDs); err != nil {
		return 0, 0, nil, err
	}
	if mode == ModeFull {
		if err := txRepo.AddSeenSourcePatchIDs(ctx, runID, patchIDs); err != nil {
			return 0, 0, nil, err
		}
	}
	commitCtx, cancel := optionalTimeoutContext(ctx, s.cfg.SyncDatabasePool.QueryTimeout)
	err = tx.Commit(commitCtx)
	cancel()
	if err != nil {
		return 0, 0, nil, err
	}
	return len(items), len(items), sourceMax, nil
}

func (s *Service) markDeletedNotSeen(ctx context.Context, lock *repository.SyncRunLock, runID uuid.UUID) (int, error) {
	beginCtx, cancel := optionalTimeoutContext(ctx, s.cfg.SyncDatabasePool.QueryTimeout)
	tx, err := lock.Begin(beginCtx)
	cancel()
	if err != nil {
		return 0, err
	}
	defer func() {
		rollbackCtx, cancel := s.finishContext(ctx)
		defer cancel()
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.log.Warn().Err(err).Stringer("run_id", runID).Msg("sync mark-deleted rollback failed")
		}
	}()
	txRepo := repository.NewSyncRepo(repository.WithQueryTimeout(tx, s.cfg.SyncDatabasePool.QueryTimeout))
	deleted, err := txRepo.MarkDeletedNotSeenByRun(ctx, runID)
	if err != nil {
		return 0, err
	}
	commitCtx, cancel := optionalTimeoutContext(ctx, s.cfg.SyncDatabasePool.QueryTimeout)
	err = tx.Commit(commitCtx)
	cancel()
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *Service) cleanupSeenSourcePatchIDs(ctx context.Context, lock *repository.SyncRunLock, runID uuid.UUID) {
	cleanupCtx, cancel := s.finishContext(ctx)
	defer cancel()
	repo := lock.Repo()
	if err := repo.CleanupSeenSourcePatchIDs(cleanupCtx, runID); err != nil {
		s.log.Warn().Err(err).Stringer("run_id", runID).Msg("sync seen staging cleanup failed")
	}
}

func (s *Service) incrementalSince(ctx context.Context, mode string, repo runStore) (*time.Time, error) {
	if mode == ModeFull {
		return nil, nil
	}
	last, err := repo.LastSuccessMaxUpdatedAt(ctx)
	if err != nil || last == nil {
		return last, err
	}
	since := last.Add(-time.Duration(s.cfg.SyncIncrementalSafetyMinutes) * time.Minute)
	return &since, nil
}

func (s *Service) querySource(ctx context.Context, sql string, args ...any) (pgx.Rows, context.CancelFunc, error) {
	queryCtx, cancel := optionalTimeoutContext(ctx, s.cfg.SourceDatabasePool.QueryTimeout)
	rows, err := s.source.Query(queryCtx, sql, args...)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return rows, cancel, nil
}

func (s *Service) querySourceGamePage(ctx context.Context, mode string, since *time.Time, afterID, limit int) ([]SourceGame, error) {
	var rows pgx.Rows
	var cancel context.CancelFunc
	var err error
	if mode == ModeIncremental && since != nil {
		rows, cancel, err = s.querySource(ctx, incrementalSourceGamesPageSQL, *since, afterID, limit)
	} else {
		rows, cancel, err = s.querySource(ctx, fullSourceGamesPageSQL, afterID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer rows.Close()
	items := make([]SourceGame, 0)
	for rows.Next() {
		var g SourceGame
		if err := rows.Scan(&g.ID, &g.UniqueID, &g.Name, &g.Introduction, &g.Banner, &g.Released, &g.ContentLimit, &g.Types, &g.Languages, &g.Platforms, &g.CreatedAt, &g.UpdatedAt, &g.ResourceUpdatedAt); err != nil {
			return nil, err
		}
		if cap(items) == 0 {
			items = make([]SourceGame, 0, limit)
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

	rows, cancel, err := s.querySource(ctx, sourceAliasesByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var value string
		if err := rows.Scan(&patchID, &value); err != nil {
			rows.Close()
			cancel()
			return relations, err
		}
		relations.aliases[patchID] = append(relations.aliases[patchID], value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		cancel()
		return relations, err
	}
	rows.Close()
	cancel()

	rows, cancel, err = s.querySource(ctx, sourceTagsByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var item model.TagData
		if err := rows.Scan(&patchID, &item.Name, &item.Aliases, &item.Source); err != nil {
			rows.Close()
			cancel()
			return relations, err
		}
		relations.tags[patchID] = append(relations.tags[patchID], item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		cancel()
		return relations, err
	}
	rows.Close()
	cancel()

	rows, cancel, err = s.querySource(ctx, sourceCompaniesByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var item model.CompanyData
		if err := rows.Scan(&patchID, &item.Name, &item.Aliases, &item.OfficialWebsites, &item.PrimaryLanguages, &item.ParentBrands); err != nil {
			rows.Close()
			cancel()
			return relations, err
		}
		relations.companies[patchID] = append(relations.companies[patchID], item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		cancel()
		return relations, err
	}
	rows.Close()
	cancel()

	rows, cancel, err = s.querySource(ctx, sourceRatingsByPatchIDsSQL, patchIDs)
	if err != nil {
		return relations, err
	}
	for rows.Next() {
		var patchID int
		var r model.RatingData
		var o1, o2, o3, o4, o5, o6, o7, o8, o9, o10 int
		if err := rows.Scan(&patchID, &r.AverageOverall, &r.Count, &r.RecStrongNo, &r.RecNo, &r.RecNeutral, &r.RecYes, &r.RecStrongYes, &o1, &o2, &o3, &o4, &o5, &o6, &o7, &o8, &o9, &o10); err != nil {
			rows.Close()
			cancel()
			return relations, err
		}
		r.Histogram = model.RatingHistogram{Score1: o1, Score2: o2, Score3: o3, Score4: o4, Score5: o5, Score6: o6, Score7: o7, Score8: o8, Score9: o9, Score10: o10}
		relations.ratings[patchID] = &r
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		cancel()
		return relations, err
	}
	rows.Close()
	cancel()

	return relations, nil
}

func (s *Service) EnsureSourceReadOnly(ctx context.Context) error {
	if s.source == nil {
		return errors.New("source database is not configured")
	}
	return ensureSourceReadOnly(ctx, s.cfg.SourceDatabasePool, s.source, s.log)
}

func ensureSourceReadOnly(ctx context.Context, cfg config.PostgresConfig, source sourceReadOnlyQueryer, log zerolog.Logger) error {
	if source == nil {
		return errors.New("source database is not configured")
	}
	queryCtx, cancel := sourceReadOnlyCheckContext(ctx, cfg)
	defer cancel()

	var transactionReadOnly string
	var databaseWritable, databaseTemporary, supportsMaintain bool
	if err := source.QueryRow(queryCtx, sourceReadOnlyStatusSQL).Scan(&transactionReadOnly, &databaseWritable, &databaseTemporary, &supportsMaintain); err != nil {
		return fmt.Errorf("check source read-only status: %w", err)
	}
	if databaseWritable {
		return errors.New("SOURCE_DATABASE_DSN user has CREATE privilege on source database")
	}
	if databaseTemporary {
		return errors.New("SOURCE_DATABASE_DSN user has TEMPORARY privilege on source database")
	}

	var tableName, relation string
	var missing, canSelect, tableWritable, schemaWritable bool
	err := source.QueryRow(queryCtx, sourceWritePrivilegeCheckSQL, sourceReadOnlyTables[:], supportsMaintain).Scan(&tableName, &relation, &missing, &canSelect, &tableWritable, &schemaWritable)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("transaction_read_only", transactionReadOnly).Msg("source read-only permissions verified")
			return nil
		}
		return fmt.Errorf("check source table privileges: %w", err)
	}
	if missing {
		return fmt.Errorf("source table %q is not visible to SOURCE_DATABASE_DSN user", tableName)
	}
	if !canSelect {
		return fmt.Errorf("SOURCE_DATABASE_DSN user does not have SELECT privilege on source table %s", relation)
	}
	if tableWritable {
		return fmt.Errorf("SOURCE_DATABASE_DSN user has write privilege on source table %s", relation)
	}
	if schemaWritable {
		return fmt.Errorf("SOURCE_DATABASE_DSN user has CREATE privilege on source schema for table %s", relation)
	}
	return errors.New("source read-only check returned an unknown privilege violation")
}

func (s *Service) String() string {
	return fmt.Sprintf("sync interval=%dm full=%dh", s.cfg.SyncIntervalMinutes, s.cfg.SyncFullIntervalHours)
}
