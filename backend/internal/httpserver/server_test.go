package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/repository"
	"github.com/touchgal/developer/backend/internal/services/auth"
)

func TestSessionAuthDoesNotRunOnPublicHealth(t *testing.T) {
	cfg := routerTestConfig()
	store := &routerSessionStore{err: model.ErrNotFound}
	router := NewRouter(cfg, Services{Auth: auth.NewService(cfg, nil, store, nil, nil)}, &repository.Repositories{}, nil, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: "raw-session"})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected health status 200, got %d", res.Code)
	}
	if store.lookups != 0 {
		t.Fatalf("public health must not perform session lookup, got %d", store.lookups)
	}
}

func TestSessionAuthDoesNotRunOnPublicReadiness(t *testing.T) {
	cfg := routerTestConfig()
	store := &routerSessionStore{err: model.ErrNotFound}
	router := NewRouter(cfg, Services{Auth: auth.NewService(cfg, nil, store, nil, nil)}, &repository.Repositories{}, nil, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: "raw-session"})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness status 503 without configured checks, got %d", res.Code)
	}
	if store.lookups != 0 {
		t.Fatalf("public readiness must not perform session lookup, got %d", store.lookups)
	}
}

func TestSessionAuthRunsOnPortalGroup(t *testing.T) {
	cfg := routerTestConfig()
	store := &routerSessionStore{err: model.ErrNotFound}
	router := NewRouter(cfg, Services{Auth: auth.NewService(cfg, nil, store, nil, nil)}, &repository.Repositories{}, nil, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: "raw-session"})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected portal route to stop at auth, got %d", res.Code)
	}
	if store.lookups != 1 {
		t.Fatalf("portal route must perform one session lookup, got %d", store.lookups)
	}
}

func TestSessionAuthRunsOnAuthMe(t *testing.T) {
	cfg := routerTestConfig()
	userID := uuid.New()
	store := &routerSessionStore{
		session: model.Session{ID: uuid.New(), UserID: userID, ExpiresAt: time.Now().Add(time.Hour)},
		user:    model.User{ID: userID, Email: "dev@example.com", DisplayName: "Dev", Status: model.UserStatusDisabled, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	router := NewRouter(cfg, Services{Auth: auth.NewService(cfg, nil, store, nil, nil)}, &repository.Repositories{}, nil, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: "raw-session"})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected /auth/me status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if store.lookups != 1 {
		t.Fatalf("/auth/me must perform one session lookup, got %d", store.lookups)
	}
}

func routerTestConfig() config.Config {
	return config.Config{
		PublicURL:                         "https://portal.example",
		SessionSecret:                     "secret",
		SessionCookieName:                 "tgal_dev_session",
		SessionAuthCacheTTLSeconds:        60,
		SessionLastSeenUpdateIntervalSecs: 300,
	}
}

type routerSessionStore struct {
	session model.Session
	user    model.User
	err     error
	lookups int
}

func (s *routerSessionStore) InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error) {
	return nil, model.ErrInvalidInput
}

func (s *routerSessionStore) LatestCode(ctx context.Context, email, purpose string) (*model.EmailVerificationCode, error) {
	return nil, model.ErrNotFound
}

func (s *routerSessionStore) IncrementCodeAttempts(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *routerSessionStore) ConsumeCode(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *routerSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, sessionHash, userAgent, ip string, expiresAt time.Time) (*model.Session, error) {
	return nil, model.ErrInvalidInput
}

func (s *routerSessionStore) GetSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.Session, *model.User, error) {
	s.lookups++
	if s.err != nil {
		return nil, nil, s.err
	}
	session := s.session
	session.SessionHash = sessionHash
	user := s.user
	return &session, &user, nil
}

func (s *routerSessionStore) TouchSessionLastSeen(ctx context.Context, id uuid.UUID, now, cutoff time.Time) error {
	return nil
}

func (s *routerSessionStore) RevokeSession(ctx context.Context, sessionHash string) error {
	return nil
}
