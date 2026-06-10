package requestlog

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeRequestLogStore struct {
	inserted      chan []model.RequestLog
	rollupCleanup chan time.Time
}

func (s *fakeRequestLogStore) InsertRequestLogs(ctx context.Context, logs []model.RequestLog) error {
	copyLogs := append([]model.RequestLog(nil), logs...)
	s.inserted <- copyLogs
	return nil
}

func (s *fakeRequestLogStore) DeleteRequestLogsBefore(ctx context.Context, before time.Time, limit int) (int, error) {
	return 0, nil
}
func (s *fakeRequestLogStore) DeleteRequestLogRollupsBefore(ctx context.Context, before time.Time, limit int) (int, error) {
	if s.rollupCleanup != nil {
		s.rollupCleanup <- before
	}
	return 0, nil
}

func TestWriterDropsWhenQueueIsFull(t *testing.T) {
	store := &fakeRequestLogStore{inserted: make(chan []model.RequestLog, 1)}
	writer := NewWriter(store, Config{QueueSize: 1, BatchSize: 10, FlushInterval: time.Hour}, zerolog.Nop())

	if !writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/me"}) {
		t.Fatal("first enqueue should fit in the bounded queue")
	}
	if writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/games/search"}) {
		t.Fatal("second enqueue should be dropped when queue is full")
	}
	if dropped := writer.Dropped(); dropped != 1 {
		t.Fatalf("dropped logs got %d", dropped)
	}
}

func TestWriterFlushesAtBatchSize(t *testing.T) {
	store := &fakeRequestLogStore{inserted: make(chan []model.RequestLog, 1)}
	writer := NewWriter(store, Config{QueueSize: 4, BatchSize: 2, FlushInterval: time.Hour}, zerolog.Nop())
	writer.Start()
	defer stopWriter(t, writer)

	writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/me"})
	writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/games/search"})

	logs := waitForInserted(t, store.inserted)
	if len(logs) != 2 {
		t.Fatalf("batch size got %d", len(logs))
	}
}

func TestWriterStopDrainsPendingLogs(t *testing.T) {
	store := &fakeRequestLogStore{inserted: make(chan []model.RequestLog, 1)}
	writer := NewWriter(store, Config{QueueSize: 4, BatchSize: 10, FlushInterval: time.Hour}, zerolog.Nop())
	writer.Start()

	writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/me"})
	writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/games/search"})
	stopWriter(t, writer)

	logs := waitForInserted(t, store.inserted)
	if len(logs) != 2 {
		t.Fatalf("drained logs got %d", len(logs))
	}
}
func TestWriterStopRejectsLaterEnqueue(t *testing.T) {
	store := &fakeRequestLogStore{inserted: make(chan []model.RequestLog, 1)}
	writer := NewWriter(store, Config{QueueSize: 4, BatchSize: 10, FlushInterval: time.Hour}, zerolog.Nop())
	writer.Start()
	stopWriter(t, writer)

	if writer.EnqueueRequestLog(model.RequestLog{Path: "/v1/me"}) {
		t.Fatal("enqueue after stop must be rejected")
	}
}

func TestWriterCleansRollups(t *testing.T) {
	store := &fakeRequestLogStore{
		inserted:      make(chan []model.RequestLog, 1),
		rollupCleanup: make(chan time.Time, 1),
	}
	writer := NewWriter(store, Config{QueueSize: 1, BatchSize: 1, FlushInterval: time.Hour}, zerolog.Nop())
	writer.Start()
	defer stopWriter(t, writer)

	select {
	case <-store.rollupCleanup:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for rollup cleanup")
	}
}

func stopWriter(t *testing.T, writer *Writer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := writer.Stop(ctx); err != nil {
		t.Fatalf("stop writer: %v", err)
	}
}

func waitForInserted(t *testing.T, inserted <-chan []model.RequestLog) []model.RequestLog {
	t.Helper()
	select {
	case logs := <-inserted:
		return logs
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inserted logs")
		return nil
	}
}
