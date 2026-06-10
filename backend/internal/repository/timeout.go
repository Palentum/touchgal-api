package repository

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func WithQueryTimeout(db Queryer, timeout time.Duration) Queryer {
	if timeout <= 0 {
		return db
	}
	return timeoutQueryer{db: db, timeout: timeout}
}

type timeoutQueryer struct {
	db      Queryer
	timeout time.Duration
}

func (q timeoutQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()
	return q.db.Exec(ctx, sql, arguments...)
}

func (q timeoutQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	rows, err := q.db.Query(ctx, sql, args...)
	if err != nil {
		cancel()
		return nil, err
	}
	return &timeoutRows{Rows: rows, cancel: cancel}, nil
}

func (q timeoutQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	return timeoutRow{row: q.db.QueryRow(ctx, sql, args...), cancel: cancel}
}

type timeoutRows struct {
	pgx.Rows
	cancel context.CancelFunc
	once   sync.Once
}

func (r *timeoutRows) Close() {
	r.Rows.Close()
	r.once.Do(r.cancel)
}

func (r *timeoutRows) Next() bool {
	ok := r.Rows.Next()
	if !ok {
		r.once.Do(r.cancel)
	}
	return ok
}

func (r *timeoutRows) Err() error {
	err := r.Rows.Err()
	if err != nil {
		r.once.Do(r.cancel)
	}
	return err
}

type timeoutRow struct {
	row    pgx.Row
	cancel context.CancelFunc
}

func (r timeoutRow) Scan(dest ...any) error {
	defer r.cancel()
	return r.row.Scan(dest...)
}
