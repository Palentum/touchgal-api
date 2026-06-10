package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errQueryerCannotBegin = errors.New("queryer does not support transactions")

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
func (q timeoutQueryer) Begin(ctx context.Context) (pgx.Tx, error) {
	beginner, ok := q.db.(txBeginner)
	if !ok {
		return nil, errQueryerCannotBegin
	}
	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return timeoutTx{Tx: tx, timeout: q.timeout}, nil
}

type timeoutTx struct {
	pgx.Tx
	timeout time.Duration
}

func (tx timeoutTx) Begin(ctx context.Context) (pgx.Tx, error) {
	ctx, cancel := context.WithTimeout(ctx, tx.timeout)
	defer cancel()
	child, err := tx.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return timeoutTx{Tx: child, timeout: tx.timeout}, nil
}

func (tx timeoutTx) Commit(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, tx.timeout)
	defer cancel()
	return tx.Tx.Commit(ctx)
}

func (tx timeoutTx) Rollback(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, tx.timeout)
	defer cancel()
	return tx.Tx.Rollback(ctx)
}

func (tx timeoutTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(ctx, tx.timeout)
	defer cancel()
	return tx.Tx.Exec(ctx, sql, arguments...)
}

func (tx timeoutTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, tx.timeout)
	rows, err := tx.Tx.Query(ctx, sql, args...)
	if err != nil {
		cancel()
		return nil, err
	}
	return &timeoutRows{Rows: rows, cancel: cancel}, nil
}

func (tx timeoutTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := context.WithTimeout(ctx, tx.timeout)
	return timeoutRow{row: tx.Tx.QueryRow(ctx, sql, args...), cancel: cancel}
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
