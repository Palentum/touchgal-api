package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakeSyncRunLockConn struct {
	row      pgx.Row
	released bool
	hijacked bool
}

func (c *fakeSyncRunLockConn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.row
}

func (c *fakeSyncRunLockConn) Release() {
	c.released = true
}

func (c *fakeSyncRunLockConn) Hijack() *pgx.Conn {
	c.hijacked = true
	return nil
}

type syncUnlockRow struct {
	unlocked bool
	err      error
}

func (r syncUnlockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*bool)) = r.unlocked
	return nil
}

func TestReleaseSyncRunLockConnReleasesAfterSuccessfulUnlock(t *testing.T) {
	conn := &fakeSyncRunLockConn{row: syncUnlockRow{unlocked: true}}

	if err := releaseSyncRunLockConn(context.Background(), conn, time.Second); err != nil {
		t.Fatalf("release sync lock: %v", err)
	}
	if !conn.released {
		t.Fatal("successful unlock must release the connection to the pool")
	}
	if conn.hijacked {
		t.Fatal("successful unlock must not destroy the connection")
	}
}

func TestReleaseSyncRunLockConnDestroysAfterUnlockError(t *testing.T) {
	unlockErr := errors.New("unlock failed")
	conn := &fakeSyncRunLockConn{row: syncUnlockRow{err: unlockErr}}

	err := releaseSyncRunLockConn(context.Background(), conn, time.Second)
	if !errors.Is(err, unlockErr) {
		t.Fatalf("expected unlock error, got %v", err)
	}
	if conn.released {
		t.Fatal("failed unlock must not return the connection to the pool")
	}
	if !conn.hijacked {
		t.Fatal("failed unlock must destroy the connection")
	}
}

func TestReleaseSyncRunLockConnDestroysWhenLockMissing(t *testing.T) {
	conn := &fakeSyncRunLockConn{row: syncUnlockRow{unlocked: false}}

	err := releaseSyncRunLockConn(context.Background(), conn, time.Second)
	if err == nil {
		t.Fatal("expected missing advisory lock error")
	}
	if conn.released {
		t.Fatal("missing lock must not return the connection to the pool")
	}
	if !conn.hijacked {
		t.Fatal("missing lock must destroy the connection")
	}
}
