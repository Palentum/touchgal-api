package requestlog

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/model"
)

const (
	retentionDeleteLimit = 10000
	rollupRetentionDays  = 90
)

type Store interface {
	InsertRequestLogs(ctx context.Context, logs []model.RequestLog) error
	DeleteRequestLogsBefore(ctx context.Context, before time.Time, limit int) (int, error)
	DeleteRequestLogRollupsBefore(ctx context.Context, before time.Time, limit int) (int, error)
}

type Config struct {
	QueueSize       int
	BatchSize       int
	FlushInterval   time.Duration
	WriteTimeout    time.Duration
	RetentionDays   int
	CleanupInterval time.Duration
}

type Writer struct {
	store   Store
	logger  zerolog.Logger
	cfg     Config
	input   chan model.RequestLog
	stop    chan struct{}
	done    chan struct{}
	started atomic.Bool
	stopped atomic.Bool
	dropped atomic.Uint64
}

func NewWriter(store Store, cfg Config, logger zerolog.Logger) *Writer {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 16384
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 2 * time.Second
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = time.Hour
	}
	return &Writer{
		store:  store,
		logger: logger,
		cfg:    cfg,
		input:  make(chan model.RequestLog, cfg.QueueSize),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (w *Writer) Start() {
	if w == nil || !w.started.CompareAndSwap(false, true) {
		return
	}
	go w.run()
}

func (w *Writer) EnqueueRequestLog(log model.RequestLog) bool {
	if w == nil || w.stopped.Load() {
		return false
	}
	select {
	case w.input <- log:
		return true
	default:
		w.dropped.Add(1)
		return false
	}
}

func (w *Writer) Stop(ctx context.Context) error {
	if w == nil || !w.started.Load() {
		return nil
	}
	if w.stopped.CompareAndSwap(false, true) {
		close(w.stop)
	}
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Writer) Dropped() uint64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

func (w *Writer) run() {
	defer close(w.done)
	batch := make([]model.RequestLog, 0, w.cfg.BatchSize)
	flushTicker := time.NewTicker(w.cfg.FlushInterval)
	defer flushTicker.Stop()
	cleanupTicker := time.NewTicker(w.cfg.CleanupInterval)
	defer cleanupTicker.Stop()

	w.cleanup()
	w.cleanupRollups()
	for {
		select {
		case log := <-w.input:
			batch = append(batch, log)
			if len(batch) >= w.cfg.BatchSize {
				batch = w.flush(batch)
			}
		case <-flushTicker.C:
			batch = w.flush(batch)
			w.logDropped()
		case <-cleanupTicker.C:
			w.cleanup()
			w.cleanupRollups()
		case <-w.stop:
			for {
				select {
				case log := <-w.input:
					batch = append(batch, log)
					if len(batch) >= w.cfg.BatchSize {
						batch = w.flush(batch)
					}
				default:
					w.flush(batch)
					w.logDropped()
					return
				}
			}
		}
	}
}

func (w *Writer) flush(batch []model.RequestLog) []model.RequestLog {
	if len(batch) == 0 {
		return batch[:0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.cfg.WriteTimeout)
	defer cancel()
	if err := w.store.InsertRequestLogs(ctx, batch); err != nil {
		w.logger.Warn().Err(err).Int("count", len(batch)).Msg("api request log batch write failed")
	}
	return batch[:0]
}

func (w *Writer) cleanup() {
	if w.cfg.RetentionDays <= 0 {
		return
	}
	before := time.Now().AddDate(0, 0, -w.cfg.RetentionDays)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		deleted, err := w.store.DeleteRequestLogsBefore(ctx, before, retentionDeleteLimit)
		if err != nil {
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) && !errors.Is(ctx.Err(), context.Canceled) {
				w.logger.Warn().Err(err).Msg("api request log retention cleanup failed")
			}
			return
		}
		if deleted == 0 {
			return
		}
		if deleted < retentionDeleteLimit {
			w.logger.Debug().Int("deleted", deleted).Msg("api request log retention cleanup completed")
			return
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (w *Writer) cleanupRollups() {
	before := time.Now().AddDate(0, 0, -rollupRetentionDays)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		deleted, err := w.store.DeleteRequestLogRollupsBefore(ctx, before, retentionDeleteLimit)
		if err != nil {
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) && !errors.Is(ctx.Err(), context.Canceled) {
				w.logger.Warn().Err(err).Msg("api request log rollup cleanup failed")
			}
			return
		}
		if deleted == 0 {
			return
		}
		if deleted < retentionDeleteLimit {
			w.logger.Debug().Int("deleted", deleted).Msg("api request log rollup cleanup completed")
			return
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (w *Writer) logDropped() {
	count := w.dropped.Swap(0)
	if count == 0 {
		return
	}
	w.logger.Warn().Uint64("dropped", count).Msg("api request logs dropped because queue is full")
}
