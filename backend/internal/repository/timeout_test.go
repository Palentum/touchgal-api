package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestWithQueryTimeoutSetsAndCancelsExecContext(t *testing.T) {
	queryer := &recordingTimeoutQueryer{}
	wrapped := WithQueryTimeout(queryer, time.Minute)

	if _, err := wrapped.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if _, ok := queryer.execCtx.Deadline(); !ok {
		t.Fatal("expected Exec context deadline")
	}
	if queryer.execCtx.Err() != context.Canceled {
		t.Fatalf("expected Exec context to be canceled after return, got %v", queryer.execCtx.Err())
	}
}

func TestWithQueryTimeoutCancelsQueryContextOnClose(t *testing.T) {
	queryer := &recordingTimeoutQueryer{rows: &fakeTimeoutRows{remaining: 1}}
	wrapped := WithQueryTimeout(queryer, time.Minute)

	rows, err := wrapped.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if _, ok := queryer.queryCtx.Deadline(); !ok {
		t.Fatal("expected Query context deadline")
	}
	if err := queryer.queryCtx.Err(); err != nil {
		t.Fatalf("query context canceled before rows close: %v", err)
	}
	rows.Close()
	if queryer.queryCtx.Err() != context.Canceled {
		t.Fatalf("expected Query context to be canceled on Close, got %v", queryer.queryCtx.Err())
	}
}

func TestWithQueryTimeoutCancelsQueryContextOnExhaustion(t *testing.T) {
	queryer := &recordingTimeoutQueryer{rows: &fakeTimeoutRows{remaining: 0}}
	wrapped := WithQueryTimeout(queryer, time.Minute)

	rows, err := wrapped.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if rows.Next() {
		t.Fatal("expected no rows")
	}
	if queryer.queryCtx.Err() != context.Canceled {
		t.Fatalf("expected Query context to be canceled when rows are exhausted, got %v", queryer.queryCtx.Err())
	}
}

func TestWithQueryTimeoutCancelsQueryRowContextAfterScan(t *testing.T) {
	queryer := &recordingTimeoutQueryer{}
	wrapped := WithQueryTimeout(queryer, time.Minute)

	if err := wrapped.QueryRow(context.Background(), "SELECT 1").Scan(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if _, ok := queryer.rowCtx.Deadline(); !ok {
		t.Fatal("expected QueryRow context deadline")
	}
	if queryer.rowCtx.Err() != context.Canceled {
		t.Fatalf("expected QueryRow context to be canceled after Scan, got %v", queryer.rowCtx.Err())
	}
}
func TestWithQueryTimeoutPreservesBegin(t *testing.T) {
	queryer := &recordingTimeoutBeginner{}
	wrapped := WithQueryTimeout(queryer, time.Minute)
	beginner, ok := wrapped.(txBeginner)
	if !ok {
		t.Fatal("wrapped queryer must preserve transaction support")
	}
	tx, err := beginner.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, ok := tx.(timeoutTx); !ok {
		t.Fatalf("expected timeout-wrapped transaction, got %T", tx)
	}
	if _, ok := queryer.beginCtx.Deadline(); !ok {
		t.Fatal("expected Begin context deadline")
	}
	if queryer.beginCtx.Err() != context.Canceled {
		t.Fatalf("expected Begin context to be canceled after return, got %v", queryer.beginCtx.Err())
	}
}

type recordingTimeoutBeginner struct {
	recordingTimeoutQueryer
	beginCtx context.Context
}

func (q *recordingTimeoutBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	q.beginCtx = ctx
	return fakeTimeoutTx{}, nil
}

type fakeTimeoutTx struct {
	pgx.Tx
}

type recordingTimeoutQueryer struct {
	execCtx  context.Context
	queryCtx context.Context
	rowCtx   context.Context
	rows     pgx.Rows
}

func (q *recordingTimeoutQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	q.execCtx = ctx
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (q *recordingTimeoutQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.queryCtx = ctx
	if q.rows == nil {
		return nil, errors.New("missing rows")
	}
	return q.rows, nil
}

func (q *recordingTimeoutQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.rowCtx = ctx
	return timeoutContextRow{ctx: ctx}
}

type timeoutContextRow struct {
	ctx context.Context
}

func (r timeoutContextRow) Scan(dest ...any) error {
	if _, ok := r.ctx.Deadline(); !ok {
		return errors.New("missing deadline")
	}
	if err := r.ctx.Err(); err != nil {
		return err
	}
	return nil
}

type fakeTimeoutRows struct {
	remaining int
	closed    bool
}

func (r *fakeTimeoutRows) Close() {
	r.closed = true
}

func (r *fakeTimeoutRows) Err() error {
	return nil
}

func (r *fakeTimeoutRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 1")
}

func (r *fakeTimeoutRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeTimeoutRows) Next() bool {
	if r.remaining <= 0 {
		r.closed = true
		return false
	}
	r.remaining--
	return true
}

func (r *fakeTimeoutRows) Scan(dest ...any) error {
	return nil
}

func (r *fakeTimeoutRows) Values() ([]any, error) {
	return nil, nil
}

func (r *fakeTimeoutRows) RawValues() [][]byte {
	return nil
}

func (r *fakeTimeoutRows) Conn() *pgx.Conn {
	return nil
}
