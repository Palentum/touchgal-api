package repository

import (
	"context"
	"errors"
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

type tokenAuthInfoRow struct {
	info model.TokenAuthInfo
	err  error
}

func (r tokenAuthInfoRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 21 {
		return errors.New("unexpected token auth scan destination count")
	}
	if err := (tokenRow{token: r.info.Token}).Scan(dest[:13]...); err != nil {
		return err
	}
	*(dest[13].(*string)) = r.info.ApplicationStatus
	*(dest[14].(*uuid.UUID)) = r.info.ApplicationID
	*(dest[15].(*uuid.UUID)) = r.info.UserID
	*(dest[16].(*string)) = r.info.UserStatus
	*(dest[17].(*int)) = r.info.UserMinuteLimit
	*(dest[18].(*int)) = r.info.UserDailyLimit
	*(dest[19].(*int)) = r.info.ApplicationMinuteLimit
	*(dest[20].(*int)) = r.info.ApplicationDailyLimit
	return nil
}

type tokenCreateBeginner struct {
	tx *tokenCreateTx
}

func (q *tokenCreateBeginner) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	panic("unexpected exec")
}

func (q *tokenCreateBeginner) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *tokenCreateBeginner) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("unexpected query row outside transaction")
}

func (q *tokenCreateBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	return q.tx, nil
}

type tokenCreateTx struct {
	pgx.Tx
	lockSQL    string
	lockArgs   []any
	createSQL  string
	createArgs []any
	row        pgx.Row
	committed  bool
}

func (tx *tokenCreateTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "SELECT id FROM users WHERE id = $1 FOR UPDATE") {
		tx.lockSQL = sql
		tx.lockArgs = args
		return uuidRow{id: args[0].(uuid.UUID)}
	}
	tx.createSQL = sql
	tx.createArgs = args
	return tx.row
}

func (tx *tokenCreateTx) Commit(ctx context.Context) error {
	tx.committed = true
	return nil
}

func (tx *tokenCreateTx) Rollback(ctx context.Context) error {
	return nil
}

type uuidRow struct {
	id uuid.UUID
}

func (r uuidRow) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = r.id
	return nil
}

func TestTokenRepoCreateLocksUserAndPassesActiveLimit(t *testing.T) {
	userID := uuid.New()
	tokenID := uuid.New()
	applicationID := uuid.New()
	tx := &tokenCreateTx{row: tokenRow{token: model.APIToken{
		ID: tokenID, UserID: userID, ApplicationID: applicationID, Name: "prod",
		TokenPrefix: "tgal_live_abcd", TokenHash: "hash", Status: model.TokenActive,
		MinuteLimit: 60, DailyLimit: 1000, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}}
	repo := NewTokenRepo(&tokenCreateBeginner{tx: tx})

	created, err := repo.Create(context.Background(), model.APIToken{
		ID: tokenID, UserID: userID, ApplicationID: applicationID, Name: "prod",
		TokenPrefix: "tgal_live_abcd", TokenHash: "hash", MinuteLimit: 60, DailyLimit: 1000,
	}, 10)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if created.ID != tokenID || created.UserID != userID || created.ApplicationID != applicationID {
		t.Fatalf("unexpected token: %#v", created)
	}
	if !strings.Contains(tx.lockSQL, "SELECT id FROM users WHERE id = $1 FOR UPDATE") {
		t.Fatalf("create must lock the user row before counting active tokens: %q", tx.lockSQL)
	}
	if !strings.Contains(tx.createSQL, "FROM (SELECT id FROM users WHERE id = $2 FOR UPDATE) AS u") {
		t.Fatalf("create must lock the user row inside insert statement: %q", tx.createSQL)
	}
	if !strings.Contains(tx.createSQL, "t.status = 'active'") || !strings.Contains(tx.createSQL, "t.expires_at IS NULL OR t.expires_at > now()") {
		t.Fatalf("create must count active non-expired tokens: %q", tx.createSQL)
	}
	if len(tx.createArgs) != 10 || tx.createArgs[0] != tokenID || tx.createArgs[1] != userID || tx.createArgs[9] != 10 {
		t.Fatalf("unexpected create args: %#v", tx.createArgs)
	}
	if !tx.committed {
		t.Fatal("create transaction must commit after inserting token")
	}
}

func TestTokenRepoCreateMapsActiveLimitToSentinel(t *testing.T) {
	tx := &tokenCreateTx{row: tokenRow{err: pgx.ErrNoRows}}
	repo := NewTokenRepo(&tokenCreateBeginner{tx: tx})

	_, err := repo.Create(context.Background(), model.APIToken{ID: uuid.New(), UserID: uuid.New()}, 1)
	if err != model.ErrTokenLimitExceeded {
		t.Fatalf("expected token limit error, got %v", err)
	}
}
func TestTokenRepoCreateRequiresTransactionCapableQueryer(t *testing.T) {
	repo := NewTokenRepo(&queryRowOnlyQueryer{row: tokenRow{}})

	_, err := repo.Create(context.Background(), model.APIToken{ID: uuid.New(), UserID: uuid.New()}, 1)
	if !errors.Is(err, errTokenCreateRequiresTransaction) {
		t.Fatalf("expected transaction-capable queryer error, got %v", err)
	}
}

func TestTokenRepoGetByHashWithApplicationIncludesUserStatus(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	applicationID := uuid.New()
	now := time.Now()
	queryer := &queryRowOnlyQueryer{row: tokenAuthInfoRow{info: model.TokenAuthInfo{
		Token: model.APIToken{
			ID:            tokenID,
			UserID:        userID,
			ApplicationID: applicationID,
			Name:          "prod",
			TokenPrefix:   "tgal_live_abcd",
			TokenHash:     "hash",
			Status:        model.TokenActive,
			MinuteLimit:   60,
			DailyLimit:    1000,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		ApplicationStatus:      model.ApplicationApproved,
		ApplicationID:          applicationID,
		UserID:                 userID,
		UserStatus:             model.UserStatusActive,
		UserMinuteLimit:        30,
		UserDailyLimit:         900,
		ApplicationMinuteLimit: 50,
		ApplicationDailyLimit:  1000,
	}}}
	repo := NewTokenRepo(queryer)

	info, err := repo.GetByHashWithApplication(context.Background(), "hash")
	if err != nil {
		t.Fatalf("get token auth info: %v", err)
	}
	if info.UserStatus != model.UserStatusActive {
		t.Fatalf("expected user status active, got %q", info.UserStatus)
	}
	if info.UserID != userID || info.ApplicationID != applicationID || info.Token.ID != tokenID {
		t.Fatalf("unexpected token auth info: %#v", info)
	}
	if !strings.Contains(queryer.sql, "u.status") {
		t.Fatalf("token auth query must include user status: %q", queryer.sql)
	}
	if len(queryer.args) != 1 || queryer.args[0] != "hash" {
		t.Fatalf("unexpected auth lookup args: %#v", queryer.args)
	}
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
