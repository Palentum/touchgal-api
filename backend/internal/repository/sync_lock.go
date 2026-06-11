package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	syncRunAdvisoryLockKey  int64 = 0x5447414c53594e43 // "TGALSYNC"
	syncRunLockCloseTimeout       = 5 * time.Second
)

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

func (l *SyncRunLock) Release(ctx context.Context) error {
	if l == nil || l.conn == nil {
		return nil
	}
	conn := l.conn
	l.conn = nil
	return releaseSyncRunLockConn(ctx, conn, l.timeout)
}

type syncRunLockReleaser interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Release()
	Hijack() *pgx.Conn
}

func releaseSyncRunLockConn(ctx context.Context, conn syncRunLockReleaser, timeout time.Duration) error {
	if conn == nil {
		return nil
	}
	unlockCtx, cancel := timeoutContext(ctx, timeout)
	defer cancel()
	var unlocked bool
	if err := conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock($1)`, syncRunAdvisoryLockKey).Scan(&unlocked); err != nil {
		destroySyncRunLockConn(conn)
		return err
	}
	if !unlocked {
		destroySyncRunLockConn(conn)
		return errors.New("sync advisory lock was not held")
	}
	conn.Release()
	return nil
}

func destroySyncRunLockConn(conn syncRunLockReleaser) {
	hijacked := conn.Hijack()
	if hijacked == nil {
		return
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), syncRunLockCloseTimeout)
	defer cancel()
	_ = hijacked.Close(closeCtx)
}

func timeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}
