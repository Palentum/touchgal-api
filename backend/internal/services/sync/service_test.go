package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/repository"
)

type fakeRunStore struct {
	mu sync.Mutex

	run        model.SyncRun
	startCount int

	startCtxErr   error
	finishCalled  chan struct{}
	finishOnce    sync.Once
	finishCtxErr  error
	finishStatus  string
	finishRelease chan struct{}
	finishMessage string
}

func newFakeRunStore() *fakeRunStore {
	return &fakeRunStore{
		run: model.SyncRun{
			ID:        uuid.New(),
			Mode:      ModeFull,
			Status:    "running",
			StartedAt: time.Now(),
		},
		finishCalled: make(chan struct{}),
	}
}

func (s *fakeRunStore) StartRun(ctx context.Context, mode string) (*model.SyncRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startCount++
	s.startCtxErr = ctx.Err()
	s.run.Mode = mode
	s.run.Status = "running"
	run := s.run
	return &run, nil
}

func (s *fakeRunStore) FinishRun(ctx context.Context, id uuid.UUID, status string, sourceMaxUpdatedAt *time.Time, seen, upserted, deleted int, message string) (*model.SyncRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finishCtxErr = ctx.Err()
	s.finishStatus = status
	s.finishMessage = message
	s.run.ID = id
	s.run.Status = status
	s.run.SourceMaxUpdatedAt = sourceMaxUpdatedAt
	s.run.GamesSeen = seen
	s.run.GamesUpserted = upserted
	s.run.GamesDeleted = deleted
	s.run.ErrorMessage = message
	run := s.run
	s.finishOnce.Do(func() { close(s.finishCalled) })
	if s.finishRelease != nil {
		<-s.finishRelease
	}
	return &run, nil
}

func (s *fakeRunStore) LastSuccessMaxUpdatedAt(ctx context.Context) (*time.Time, error) {
	return nil, nil
}

type readOnlyCheckCall struct {
	sql  string
	args []any
}

type readOnlyCheckQueryer struct {
	rows  []readOnlyCheckRow
	calls []readOnlyCheckCall
}

func (q *readOnlyCheckQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.calls = append(q.calls, readOnlyCheckCall{sql: sql, args: args})
	if len(q.rows) == 0 {
		return readOnlyCheckRow{err: errors.New("unexpected source read-only check query")}
	}
	row := q.rows[0]
	q.rows = q.rows[1:]
	return row
}

type readOnlyCheckRow struct {
	values []any
	err    error
}

func (r readOnlyCheckRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan destination count %d does not match value count %d", len(dest), len(r.values))
	}
	for i, value := range r.values {
		switch dest := dest[i].(type) {
		case *string:
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("value %d is %T, not string", i, value)
			}
			*dest = text
		case *bool:
			boolean, ok := value.(bool)
			if !ok {
				return fmt.Errorf("value %d is %T, not bool", i, value)
			}
			*dest = boolean
		default:
			return fmt.Errorf("unsupported scan destination %T", dest)
		}
	}
	return nil
}

func TestEnsureSourceReadOnlyAllowsSelectOnlySource(t *testing.T) {
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{
			{values: []any{"off", false, false, true}},
			{err: pgx.ErrNoRows},
		},
	}

	if err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop()); err != nil {
		t.Fatalf("ensure source read-only: %v", err)
	}
	if got, want := len(queryer.calls), 2; got != want {
		t.Fatalf("expected %d read-only check queries, got %d", want, got)
	}
	if queryer.calls[0].sql != sourceReadOnlyStatusSQL {
		t.Fatalf("expected source status probe first, got %q", queryer.calls[0].sql)
	}
	tables, ok := queryer.calls[1].args[0].([]string)
	if !ok {
		t.Fatalf("expected batched source table list argument, got %T", queryer.calls[1].args[0])
	}
	if got, want := len(tables), len(sourceReadOnlyTables); got != want {
		t.Fatalf("expected %d source tables in one privilege query, got %d", want, got)
	}
	supportsMaintain, ok := queryer.calls[1].args[1].(bool)
	if !ok || !supportsMaintain {
		t.Fatalf("expected PostgreSQL 17 MAINTAIN support flag, got %#v", queryer.calls[1].args[1])
	}
}

func TestEnsureSourceReadOnlyRejectsDatabaseCreatePrivilege(t *testing.T) {
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{{values: []any{"off", true, false, false}}},
	}

	err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "CREATE privilege on source database") {
		t.Fatalf("expected source database create privilege error, got %v", err)
	}
	if got, want := len(queryer.calls), 1; got != want {
		t.Fatalf("expected source table check to be skipped after database privilege failure, got %d queries", got)
	}
}

func TestEnsureSourceReadOnlyRejectsDatabaseTemporaryPrivilege(t *testing.T) {
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{{values: []any{"off", false, true, false}}},
	}

	err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "TEMPORARY privilege on source database") {
		t.Fatalf("expected source database temporary privilege error, got %v", err)
	}
	if got, want := len(queryer.calls), 1; got != want {
		t.Fatalf("expected source table check to be skipped after database privilege failure, got %d queries", got)
	}
}

func TestEnsureSourceReadOnlyRejectsWritableSourcePrivilege(t *testing.T) {
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{
			{values: []any{"off", false, false, true}},
			{values: []any{"patch", "patch", false, true, true, false}},
		},
	}

	err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "write privilege") {
		t.Fatalf("expected writable source privilege error, got %v", err)
	}
}

func TestEnsureSourceReadOnlyRejectsMaintainSourcePrivilege(t *testing.T) {
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{
			{values: []any{"off", false, false, true}},
			{values: []any{"patch", "patch", false, true, true, false}},
		},
	}

	err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "write privilege") {
		t.Fatalf("expected source maintain privilege error, got %v", err)
	}
	supportsMaintain, ok := queryer.calls[1].args[1].(bool)
	if !ok || !supportsMaintain {
		t.Fatalf("expected PostgreSQL 17 MAINTAIN support flag, got %#v", queryer.calls[1].args[1])
	}
}

func TestEnsureSourceReadOnlyRejectsSourceSchemaCreatePrivilege(t *testing.T) {
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{
			{values: []any{"off", false, false, false}},
			{values: []any{"patch", "patch", false, true, false, true}},
		},
	}

	err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop())
	if err == nil || !strings.Contains(err.Error(), "CREATE privilege on source schema") {
		t.Fatalf("expected source schema create privilege error, got %v", err)
	}
}

func TestEnsureSourceReadOnlyReturnsProbeFailure(t *testing.T) {
	probeErr := errors.New("show failed")
	queryer := &readOnlyCheckQueryer{
		rows: []readOnlyCheckRow{{err: probeErr}},
	}

	err := ensureSourceReadOnly(context.Background(), config.PostgresConfig{}, queryer, zerolog.Nop())
	if !errors.Is(err, probeErr) {
		t.Fatalf("expected probe error, got %v", err)
	}
}

func TestSourceReadOnlyCheckContextCapsLongSyncQueryTimeout(t *testing.T) {
	start := time.Now()
	ctx, cancel := sourceReadOnlyCheckContext(context.Background(), config.PostgresConfig{QueryTimeout: time.Hour})
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected read-only check context deadline")
	}
	if until := deadline.Sub(start); until > sourceReadOnlyCheckTimeout+time.Second {
		t.Fatalf("expected read-only check timeout capped near %s, got %s", sourceReadOnlyCheckTimeout, until)
	}
}

func TestRunRejectsSourceReadOnlyFailureBeforeCreatingRun(t *testing.T) {
	store := newFakeRunStore()
	checkErr := errors.New("source is writable")
	service := &Service{
		repo: store,
		checkSourceReadOnly: func(context.Context) error {
			return checkErr
		},
		log: zerolog.Nop(),
	}

	_, err := service.Run(context.Background(), ModeIncremental)
	if !errors.Is(err, checkErr) {
		t.Fatalf("expected source read-only check error, got %v", err)
	}

	store.mu.Lock()
	startCount := store.startCount
	store.mu.Unlock()
	if startCount != 0 {
		t.Fatalf("expected sync run not to be created on source check failure, got %d starts", startCount)
	}
}

func TestRunStartedPersistsFailureWithCanceledContext(t *testing.T) {
	store := newFakeRunStore()
	service := &Service{repo: store, log: zerolog.Nop()}
	run := &model.SyncRun{ID: uuid.New(), Mode: ModeFull, Status: "running", StartedAt: time.Now()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	updated, err := service.runStarted(ctx, run, nil, store)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_DATABASE_DSN") {
		t.Fatalf("expected source configuration error, got run=%#v err=%v", updated, err)
	}
	if updated == nil || updated.Status != "failed" {
		t.Fatalf("expected persisted failed run, got %#v", updated)
	}
	if store.finishCtxErr != nil {
		t.Fatalf("finish used canceled context: %v", store.finishCtxErr)
	}
	if store.finishStatus != "failed" {
		t.Fatalf("expected failed status, got %q", store.finishStatus)
	}
	if !strings.Contains(store.finishMessage, "SOURCE_DATABASE_DSN") {
		t.Fatalf("expected source error message, got %q", store.finishMessage)
	}
}
func TestRunRejectsDistributedLockContentionBeforeCreatingRun(t *testing.T) {
	store := newFakeRunStore()
	service := &Service{
		repo:   store,
		target: &pgxpool.Pool{},
		acquireLock: func(context.Context, *pgxpool.Pool, time.Duration) (*repository.SyncRunLock, bool, error) {
			return nil, false, nil
		},
		log: zerolog.Nop(),
	}

	_, err := service.Run(context.Background(), ModeIncremental)
	if !errors.Is(err, model.ErrSyncRunning) {
		t.Fatalf("expected distributed lock conflict, got %v", err)
	}

	store.mu.Lock()
	startCount := store.startCount
	store.mu.Unlock()
	if startCount != 0 {
		t.Fatalf("expected sync run not to be created on lock conflict, got %d starts", startCount)
	}
}

func TestStartReturnsRunningRunAndFinishesDetached(t *testing.T) {
	store := newFakeRunStore()
	service := &Service{repo: store, checkSourceReadOnly: func(context.Context) error { return nil }, log: zerolog.Nop()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	run, err := service.Start(ctx, ModeIncremental)
	if err != nil {
		t.Fatalf("start sync: %v", err)
	}
	if run.Status != "running" {
		t.Fatalf("expected immediate running run, got %q", run.Status)
	}
	if run.Mode != ModeIncremental {
		t.Fatalf("expected incremental mode, got %q", run.Mode)
	}
	if store.startCtxErr != nil {
		t.Fatalf("start used canceled context: %v", store.startCtxErr)
	}

	select {
	case <-store.finishCalled:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async finish")
	}
	if store.finishCtxErr != nil {
		t.Fatalf("async finish used canceled context: %v", store.finishCtxErr)
	}
	if store.finishStatus != "failed" {
		t.Fatalf("expected async run to persist failed status, got %q", store.finishStatus)
	}
	service.Stop()
}

func TestStartRejectsConcurrentRun(t *testing.T) {
	store := newFakeRunStore()
	store.finishRelease = make(chan struct{})
	service := &Service{repo: store, checkSourceReadOnly: func(context.Context) error { return nil }, log: zerolog.Nop()}

	run, err := service.Start(context.Background(), ModeFull)
	if err != nil {
		t.Fatalf("start sync: %v", err)
	}
	if run.Status != "running" {
		t.Fatalf("expected immediate running run, got %q", run.Status)
	}

	select {
	case <-store.finishCalled:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first async run to reach finish")
	}
	_, err = service.Start(context.Background(), ModeIncremental)
	if !errors.Is(err, model.ErrSyncRunning) {
		t.Fatalf("expected concurrent start conflict, got %v", err)
	}

	close(store.finishRelease)
	service.Stop()
}

func TestSourceGamesPageSQLUsesKeysetAndSplitIncrementalPredicates(t *testing.T) {
	if !strings.Contains(fullSourceGamesPageSQL, "WHERE id > $1") ||
		!strings.Contains(fullSourceGamesPageSQL, "ORDER BY id") ||
		!strings.Contains(fullSourceGamesPageSQL, "LIMIT $2") {
		t.Fatalf("full source page SQL must use keyset pagination, got %s", fullSourceGamesPageSQL)
	}
	if strings.Contains(incrementalSourceGamesPageSQL, " OR ") {
		t.Fatalf("incremental source page SQL must not use an OR predicate, got %s", incrementalSourceGamesPageSQL)
	}
	for _, want := range []string{
		"UNION",
		"updated >= $1 AND id > $2",
		"resource_update_time >= $1 AND id > $2",
		"ORDER BY p.id",
		"LIMIT $3",
	} {
		if !strings.Contains(incrementalSourceGamesPageSQL, want) {
			t.Fatalf("incremental source page SQL missing %q in %s", want, incrementalSourceGamesPageSQL)
		}
	}
}
