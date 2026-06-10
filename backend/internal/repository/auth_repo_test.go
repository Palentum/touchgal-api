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

func TestAuthRepoGetSessionUserReturnsDisabledUser(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour)
	expiresAt := time.Now().Add(time.Hour)
	queryer := &authSessionQueryer{row: authSessionRow{
		session: model.Session{
			ID:          uuid.New(),
			UserID:      uuid.New(),
			SessionHash: "hash",
			UserAgent:   "agent",
			IP:          "127.0.0.1",
			ExpiresAt:   expiresAt,
			CreatedAt:   createdAt,
		},
		user: model.User{
			ID:          uuid.New(),
			Email:       "dev@example.com",
			DisplayName: "Dev",
			Status:      model.UserStatusDisabled,
			MinuteLimit: 60,
			DailyLimit:  5000,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		},
	}}

	_, user, err := NewAuthRepo(queryer).GetSessionUser(context.Background(), "hash", time.Now())
	if err != nil {
		t.Fatalf("expected disabled session user, got error %v", err)
	}
	if user.Status != model.UserStatusDisabled {
		t.Fatalf("expected disabled user, got %q", user.Status)
	}
	if strings.Contains(queryer.sql, "u.status = 'active'") {
		t.Fatalf("session lookup must not hide disabled accounts: %s", queryer.sql)
	}
	if queryer.execCalls != 0 {
		t.Fatal("session lookup must not write last_seen_at")
	}
}

func TestAuthRepoTouchSessionLastSeenUsesCutoff(t *testing.T) {
	queryer := &authSessionQueryer{}
	sessionID := uuid.New()
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-5 * time.Minute)

	if err := NewAuthRepo(queryer).TouchSessionLastSeen(context.Background(), sessionID, now, cutoff); err != nil {
		t.Fatalf("touch session last seen: %v", err)
	}
	if queryer.execCalls != 1 {
		t.Fatalf("expected one last_seen_at update, got %d", queryer.execCalls)
	}
	if !strings.Contains(queryer.execSQL, "last_seen_at IS NULL OR last_seen_at < $3") {
		t.Fatalf("expected cutoff guard in SQL, got %s", queryer.execSQL)
	}
	if queryer.execArgs[0] != sessionID || queryer.execArgs[1] != now || queryer.execArgs[2] != cutoff {
		t.Fatalf("unexpected touch args: %#v", queryer.execArgs)
	}
}

type authSessionQueryer struct {
	sql             string
	execSQL         string
	execArgs        []any
	execCalls       int
	row             pgx.Row
	touchedLastSeen bool
}

func (q *authSessionQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	q.execSQL = sql
	q.execArgs = arguments
	q.execCalls++
	q.touchedLastSeen = strings.Contains(sql, "last_seen_at")
	return pgconn.CommandTag{}, nil
}

func (q *authSessionQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *authSessionQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.sql = sql
	return q.row
}

type authSessionRow struct {
	session model.Session
	user    model.User
	err     error
}

func (r authSessionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*uuid.UUID)) = r.session.ID
	*(dest[1].(*uuid.UUID)) = r.session.UserID
	*(dest[2].(*string)) = r.session.SessionHash
	*(dest[3].(*string)) = r.session.UserAgent
	*(dest[4].(*string)) = r.session.IP
	*(dest[5].(*time.Time)) = r.session.ExpiresAt
	*(dest[6].(**time.Time)) = r.session.RevokedAt
	*(dest[7].(*time.Time)) = r.session.CreatedAt
	*(dest[8].(**time.Time)) = r.session.LastSeenAt
	*(dest[9].(*uuid.UUID)) = r.user.ID
	*(dest[10].(*string)) = r.user.Email
	*(dest[11].(*string)) = r.user.DisplayName
	*(dest[12].(*string)) = r.user.Status
	*(dest[13].(*bool)) = r.user.IsAdmin
	*(dest[14].(*int)) = r.user.MinuteLimit
	*(dest[15].(*int)) = r.user.DailyLimit
	*(dest[16].(**time.Time)) = r.user.LastLoginAt
	*(dest[17].(*time.Time)) = r.user.CreatedAt
	*(dest[18].(*time.Time)) = r.user.UpdatedAt
	return nil
}
