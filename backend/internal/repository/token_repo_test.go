package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type execOnlyQueryer struct {
	sql  string
	args []any
	tag  pgconn.CommandTag
	err  error
}

func (q *execOnlyQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	q.sql = sql
	q.args = arguments
	return q.tag, q.err
}

func (q *execOnlyQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *execOnlyQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("unexpected query row")
}

type queryRowOnlyQueryer struct {
	sql  string
	args []any
	row  pgx.Row
}

func (q *queryRowOnlyQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	panic("unexpected exec")
}

func (q *queryRowOnlyQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *queryRowOnlyQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.sql = sql
	q.args = args
	return q.row
}

type tokenRow struct {
	token model.APIToken
	err   error
}

func (r tokenRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*uuid.UUID)) = r.token.ID
	*(dest[1].(*uuid.UUID)) = r.token.UserID
	*(dest[2].(*uuid.UUID)) = r.token.ApplicationID
	*(dest[3].(*string)) = r.token.Name
	*(dest[4].(*string)) = r.token.TokenPrefix
	*(dest[5].(*string)) = r.token.TokenHash
	*(dest[6].(*string)) = r.token.Status
	*(dest[7].(*int)) = r.token.MinuteLimit
	*(dest[8].(*int)) = r.token.DailyLimit
	*(dest[9].(**time.Time)) = r.token.LastUsedAt
	*(dest[10].(**time.Time)) = r.token.ExpiresAt
	*(dest[11].(*time.Time)) = r.token.CreatedAt
	*(dest[12].(*time.Time)) = r.token.UpdatedAt
	return nil
}

func TestTokenRepoUpdateNameForUserRenamesScopedRow(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()
	queryer := &queryRowOnlyQueryer{row: tokenRow{token: model.APIToken{
		ID: tokenID, UserID: userID, ApplicationID: uuid.New(), Name: "renamed",
		TokenPrefix: "tgal_live_abcd", TokenHash: "hash", Status: model.TokenActive,
		MinuteLimit: 60, DailyLimit: 1000, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}}}
	repo := NewTokenRepo(queryer)

	token, err := repo.UpdateNameForUser(context.Background(), tokenID, userID, "renamed")
	if err != nil {
		t.Fatalf("update token name: %v", err)
	}
	if token.Name != "renamed" || token.ID != tokenID || token.UserID != userID {
		t.Fatalf("unexpected updated token: %#v", token)
	}
	if !strings.Contains(queryer.sql, "UPDATE api_tokens") || !strings.Contains(queryer.sql, "SET name = $3") {
		t.Fatalf("expected token name update SQL, got %q", queryer.sql)
	}
	if !strings.Contains(queryer.sql, "WHERE id = $1 AND user_id = $2") {
		t.Fatalf("token rename must be scoped by token and user id: %q", queryer.sql)
	}
	if len(queryer.args) != 3 || queryer.args[0] != tokenID || queryer.args[1] != userID || queryer.args[2] != "renamed" {
		t.Fatalf("unexpected update args: %#v", queryer.args)
	}
}

func TestTokenRepoUpdateNameForUserRequiresExistingRow(t *testing.T) {
	queryer := &queryRowOnlyQueryer{row: tokenRow{err: pgx.ErrNoRows}}
	repo := NewTokenRepo(queryer)

	if _, err := repo.UpdateNameForUser(context.Background(), uuid.New(), uuid.New(), "renamed"); err != model.ErrNotFound {
		t.Fatalf("expected not found for missing token, got %v", err)
	}
}

func TestTokenRepoDeleteForUserRemovesRow(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	queryer := &execOnlyQueryer{tag: pgconn.NewCommandTag("DELETE 1")}
	repo := NewTokenRepo(queryer)

	if err := repo.DeleteForUser(context.Background(), tokenID, userID); err != nil {
		t.Fatalf("delete token for user: %v", err)
	}
	if !strings.Contains(queryer.sql, "DELETE FROM api_tokens") {
		t.Fatalf("expected token deletion SQL, got %q", queryer.sql)
	}
	if strings.Contains(queryer.sql, "UPDATE") || strings.Contains(queryer.sql, "revoked_at") {
		t.Fatalf("token invalidation must delete, not mark revoked: %q", queryer.sql)
	}
	if len(queryer.args) != 2 || queryer.args[0] != tokenID || queryer.args[1] != userID {
		t.Fatalf("delete must be scoped by token and user id, got %#v", queryer.args)
	}
}

func TestTokenRepoDeleteForUserRequiresExistingRow(t *testing.T) {
	queryer := &execOnlyQueryer{tag: pgconn.NewCommandTag("DELETE 0")}
	repo := NewTokenRepo(queryer)

	if err := repo.DeleteForUser(context.Background(), uuid.New(), uuid.New()); err != model.ErrNotFound {
		t.Fatalf("expected not found for missing token, got %v", err)
	}
}

func TestTokenRepoDeleteByIDRemovesRow(t *testing.T) {
	tokenID := uuid.New()
	queryer := &execOnlyQueryer{tag: pgconn.NewCommandTag("DELETE 1")}
	repo := NewTokenRepo(queryer)

	if err := repo.DeleteByID(context.Background(), tokenID); err != nil {
		t.Fatalf("delete token by id: %v", err)
	}
	if !strings.Contains(queryer.sql, "DELETE FROM api_tokens") {
		t.Fatalf("expected token deletion SQL, got %q", queryer.sql)
	}
	if len(queryer.args) != 1 || queryer.args[0] != tokenID {
		t.Fatalf("delete by id must pass the token id only, got %#v", queryer.args)
	}
}
