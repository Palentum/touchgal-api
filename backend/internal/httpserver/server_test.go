package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

func TestAuthStartRoutesUseIPRateLimit(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "login", path: "/auth/login/start", body: `{"email":"not-an-email","turnstileToken":""}`},
		{name: "register", path: "/auth/register/start", body: `{"email":"not-an-email","displayName":"Dev","turnstileToken":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisServer := miniredis.RunT(t)
			redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
			defer redisClient.Close()

			cfg := routerTestConfig()
			cfg.AuthCodeIPMinuteLimit = 1
			cfg.AuthCodeIPDailyLimit = 10
			router := NewRouter(cfg, Services{Auth: auth.NewService(cfg, nil, nil, redisClient, nil)}, &repository.Repositories{}, redisClient, zerolog.Nop())

			request := func() int {
				req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
				req.RemoteAddr = "203.0.113.10:12345"
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				return res.Code
			}

			if code := request(); code != http.StatusBadRequest {
				t.Fatalf("first request must reach auth handler and fail input validation, got %d", code)
			}
			if code := request(); code != http.StatusTooManyRequests {
				t.Fatalf("second request must be blocked by auth IP rate limit, got %d", code)
			}
		})
	}
}

func TestRouterAppliesSecurityHeadersToResponses(t *testing.T) {
	router := NewRouter(routerTestConfig(), Services{}, &repository.Repositories{}, nil, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "no-referrer",
		"X-Frame-Options":        "DENY",
	}
	for header, want := range checks {
		if got := res.Header().Get(header); got != want {
			t.Fatalf("expected %s %q, got %q", header, want, got)
		}
	}
	if csp := res.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("expected frame-ancestors in CSP, got %q", csp)
	}
	if permissions := res.Header().Get("Permissions-Policy"); permissions == "" {
		t.Fatal("expected Permissions-Policy header")
	}
}

func TestRouterAppliesNoStoreToSensitiveRoutes(t *testing.T) {
	router := NewRouter(routerTestConfig(), Services{}, &repository.Repositories{}, nil, zerolog.Nop())
	sensitivePaths := []string{"/auth/me", "/tokens", "/admin/users", "/dashboard/stats"}

	for _, path := range sensitivePaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			if got := res.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("expected Cache-Control no-store, got %q", got)
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if got := res.Header().Get("Cache-Control"); got == "no-store" {
		t.Fatalf("public health route must not inherit no-store, got %q", got)
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
