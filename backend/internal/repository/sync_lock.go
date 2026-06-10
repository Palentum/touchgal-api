package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const syncRunAdvisoryLockKey int64 = 0x5447414c53594e43 // "TGALSYNC"

type SyncRunLock struct {
	conn    *pgxpool.Conn
	timeout time.Duration
}

func TryAcquireSyncRunLock(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) (*SyncRunLock, bool, error) {
	acquireCtx, cancel := timeoutContext(ctx, timeout)
	conn, err := pool.Acquire(acquireCtx)
	cancel()
	if err != nil {
		return nil, false, err
	}

	queryCtx, cancel := timeoutContext(ctx, timeout)
	var locked bool
	err = conn.QueryRow(queryCtx, `SELECT pg_try_advisory_lock($1)`, syncRunAdvisoryLockKey).Scan(&locked)
	cancel()
	if err != nil {
		conn.Release()
		return nil, false, err
	}
	if !locked {
		conn.Release()
		return nil, false, nil
	}

	return &SyncRunLock{conn: conn, timeout: timeout}, true, nil
}

func (l *SyncRunLock) Begin(ctx context.Context) (pgx.Tx, error) {
	queryCtx, cancel := timeoutContext(ctx, l.timeout)
	defer cancel()
	return l.conn.Begin(queryCtx)
}

func (l *SyncRunLock) Repo() *SyncRepo {
	return NewSyncRepo(WithQueryTimeout(l.conn, l.timeout))
}

func (l *SyncRunLock) Release(ctx context.Context) {
	if l == nil || l.conn == nil {
		return
	}
	conn := l.conn
	l.conn = nil
	defer conn.Release()
	unlockCtx, cancel := timeoutContext(ctx, l.timeout)
	defer cancel()
	_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, syncRunAdvisoryLockKey)
}

func timeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}
