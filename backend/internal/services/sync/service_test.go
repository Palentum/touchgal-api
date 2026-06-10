package syncsvc

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeRunStore struct {
	mu sync.Mutex

	run model.SyncRun

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

func TestRunStartedPersistsFailureWithCanceledContext(t *testing.T) {
	store := newFakeRunStore()
	service := &Service{repo: store, log: zerolog.Nop()}
	run := &model.SyncRun{ID: uuid.New(), Mode: ModeFull, Status: "running", StartedAt: time.Now()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	updated, err := service.runStarted(ctx, run)
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

func TestStartReturnsRunningRunAndFinishesDetached(t *testing.T) {
	store := newFakeRunStore()
	service := &Service{repo: store, log: zerolog.Nop()}
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
	service := &Service{repo: store, log: zerolog.Nop()}

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
