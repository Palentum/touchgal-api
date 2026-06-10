package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestSessionTokenHash(t *testing.T) {
	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	hash := HashSessionToken(token, "secret")
	if hash == token {
		t.Fatal("hash must not equal plaintext token")
	}
	if HashSessionToken(token, "secret") != hash {
		t.Fatal("hash must be deterministic")
	}
	if HashSessionToken(token, "other") == hash {
		t.Fatal("secret must affect hash")
	}
}

func TestCurrentUserCachesSessionUserAndThrottlesLastSeen(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	raw := "raw-session-token"
	store := newSessionCacheStore(now)
	svc := newSessionCacheService(store, client, func() time.Time { return now })

	first, err := svc.CurrentUser(context.Background(), raw)
	if err != nil {
		t.Fatalf("first current user: %v", err)
	}
	second, err := svc.CurrentUser(context.Background(), raw)
	if err != nil {
		t.Fatalf("cached current user: %v", err)
	}
	if first.ID != store.user.ID || second.ID != store.user.ID {
		t.Fatalf("unexpected cached users: %#v %#v", first, second)
	}
	if store.sessionLookups != 1 {
		t.Fatalf("expected one DB session lookup, got %d", store.sessionLookups)
	}
	if store.lastSeenTouches != 1 {
		t.Fatalf("expected one throttled last_seen_at touch, got %d", store.lastSeenTouches)
	}
}

func TestCurrentUserRefreshesAfterSessionCacheTTL(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	store := newSessionCacheStore(now)
	svc := newSessionCacheService(store, client, func() time.Time { return now })
	svc.cfg.SessionAuthCacheTTLSeconds = 1

	if _, err := svc.CurrentUser(context.Background(), "raw-session-token"); err != nil {
		t.Fatalf("first current user: %v", err)
	}
	server.FastForward(2 * time.Second)
	now = now.Add(2 * time.Second)
	if _, err := svc.CurrentUser(context.Background(), "raw-session-token"); err != nil {
		t.Fatalf("refreshed current user: %v", err)
	}
	if store.sessionLookups != 2 {
		t.Fatalf("expected DB lookup after cache ttl, got %d", store.sessionLookups)
	}
}

func TestRevokeSessionDeletesSessionCache(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	raw := "raw-session-token"
	store := newSessionCacheStore(now)
	svc := newSessionCacheService(store, client, func() time.Time { return now })
	hash := HashSessionToken(raw, svc.cfg.SessionSecret)

	if _, err := svc.CurrentUser(context.Background(), raw); err != nil {
		t.Fatalf("current user: %v", err)
	}
	if exists := client.Exists(context.Background(), sessionAuthCacheKey(hash)).Val(); exists != 1 {
		t.Fatalf("expected session cache before revoke, got exists=%d", exists)
	}
	if err := svc.RevokeSession(context.Background(), raw); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if exists := client.Exists(context.Background(), sessionRevokedKey(hash)).Val(); exists != 1 {
		t.Fatalf("expected revoke tombstone after revoke, got exists=%d", exists)
	}
	if store.revokedHash != hash {
		t.Fatalf("expected revoked hash %q, got %q", hash, store.revokedHash)
	}
	if exists := client.Exists(context.Background(), sessionAuthCacheKey(hash)).Val(); exists != 0 {
		t.Fatalf("expected session cache deleted after revoke, got exists=%d", exists)
	}
}

func TestRevokedSessionTombstoneBypassesStaleCache(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	raw := "raw-session-token"
	store := newSessionCacheStore(now)
	svc := newSessionCacheService(store, client, func() time.Time { return now })
	hash := HashSessionToken(raw, svc.cfg.SessionSecret)

	if _, err := svc.CurrentUser(context.Background(), raw); err != nil {
		t.Fatalf("current user: %v", err)
	}
	if err := client.Set(context.Background(), sessionRevokedKey(hash), "1", time.Minute).Err(); err != nil {
		t.Fatalf("set revoked tombstone: %v", err)
	}
	store.err = model.ErrNotFound

	if _, err := svc.CurrentUser(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected revoked cached session to be unauthorized, got %v", err)
	}
	if store.sessionLookups != 2 {
		t.Fatalf("expected cache bypass and DB lookup after tombstone, got %d lookups", store.sessionLookups)
	}
	if exists := client.Exists(context.Background(), sessionAuthCacheKey(hash)).Val(); exists != 0 {
		t.Fatalf("expected stale session cache deleted after tombstone, got exists=%d", exists)
	}
}

func TestInvalidateUserSessionsDeletesSessionCache(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	raw := "raw-session-token"
	store := newSessionCacheStore(now)
	svc := newSessionCacheService(store, client, func() time.Time { return now })
	hash := HashSessionToken(raw, svc.cfg.SessionSecret)

	if _, err := svc.CurrentUser(context.Background(), raw); err != nil {
		t.Fatalf("current user: %v", err)
	}
	svc.InvalidateUserSessions(context.Background(), store.user.ID)
	if exists := client.Exists(context.Background(), sessionAuthCacheKey(hash)).Val(); exists != 0 {
		t.Fatalf("expected session cache deleted after user invalidation, got exists=%d", exists)
	}
	if exists := client.Exists(context.Background(), sessionUserSessionsKey(store.user.ID)).Val(); exists != 0 {
		t.Fatalf("expected user session index deleted, got exists=%d", exists)
	}
}

func newSessionCacheService(store *sessionCacheStore, client *redis.Client, nowFunc func() time.Time) *Service {
	return &Service{
		cfg: config.Config{
			SessionSecret:                     "secret",
			SessionAuthCacheTTLSeconds:        60,
			SessionLastSeenUpdateIntervalSecs: 300,
		},
		codes:   store,
		redis:   client,
		nowFunc: nowFunc,
	}
}

func newSessionCacheStore(now time.Time) *sessionCacheStore {
	userID := uuid.New()
	return &sessionCacheStore{
		session: model.Session{
			ID:          uuid.New(),
			UserID:      userID,
			SessionHash: "hash",
			ExpiresAt:   now.Add(time.Hour),
			CreatedAt:   now.Add(-time.Hour),
		},
		user: model.User{
			ID:          userID,
			Email:       "dev@example.com",
			DisplayName: "Dev",
			Status:      model.UserStatusActive,
			MinuteLimit: 60,
			DailyLimit:  5000,
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Hour),
		},
	}
}

type sessionCacheStore struct {
	session         model.Session
	user            model.User
	err             error
	sessionLookups  int
	lastSeenTouches int
	revokedHash     string
}

func (s *sessionCacheStore) InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error) {
	return nil, model.ErrInvalidInput
}

func (s *sessionCacheStore) LatestCode(ctx context.Context, email, purpose string) (*model.EmailVerificationCode, error) {
	return nil, model.ErrNotFound
}

func (s *sessionCacheStore) IncrementCodeAttempts(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *sessionCacheStore) ConsumeCode(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *sessionCacheStore) CreateSession(ctx context.Context, userID uuid.UUID, sessionHash, userAgent, ip string, expiresAt time.Time) (*model.Session, error) {
	return nil, model.ErrInvalidInput
}

func (s *sessionCacheStore) GetSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.Session, *model.User, error) {
	s.sessionLookups++
	if s.err != nil {
		return nil, nil, s.err
	}
	session := s.session
	session.SessionHash = sessionHash
	user := s.user
	return &session, &user, nil
}

func (s *sessionCacheStore) TouchSessionLastSeen(ctx context.Context, id uuid.UUID, now, cutoff time.Time) error {
	s.lastSeenTouches++
	return nil
}

func (s *sessionCacheStore) RevokeSession(ctx context.Context, sessionHash string) error {
	s.revokedHash = sessionHash
	return nil
}
