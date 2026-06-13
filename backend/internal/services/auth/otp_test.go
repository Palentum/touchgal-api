package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestGenerateAndVerifyCodeHash(t *testing.T) {
	code, err := GenerateNumericCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6 digits, got %q", code)
	}
	hash := HashCode("login", "User@Example.com", code, "secret")
	if !VerifyCodeHash("login", "user@example.com", code, "secret", hash) {
		t.Fatal("expected code hash to verify")
	}
	if VerifyCodeHash("login", "user@example.com", "000000", "secret", hash) {
		t.Fatal("wrong code verified")
	}
}

func TestVerifyLoginConcurrentReplayConsumesCodeOnce(t *testing.T) {
	const (
		emailAddr = "user@example.com"
		rawCode   = "123456"
		secret    = "test-session-secret"
	)

	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	codeID := uuid.New()
	store := newReplayCodeSessionStore(&model.EmailVerificationCode{
		ID:        codeID,
		Email:     emailAddr,
		Purpose:   "login",
		CodeHash:  HashCode("login", emailAddr, rawCode, secret),
		ExpiresAt: now.Add(5 * time.Minute),
		Attempts:  0,
		CreatedAt: now,
	})
	defer store.releaseLatest()

	user := &model.User{
		ID:          uuid.New(),
		Email:       emailAddr,
		DisplayName: "User",
		Status:      model.UserStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	service := NewService(
		config.Config{
			SessionSecret:        secret,
			SessionTTLHours:      1,
			EmailCodeMaxAttempts: 3,
		},
		disabledUserStore{user: user},
		store,
		nil,
		&disabledMailer{},
	)
	service.nowFunc = func() time.Time { return now }

	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := service.VerifyLogin(context.Background(), emailAddr, rawCode, "agent", "127.0.0.1")
			errs <- err
		}()
	}

	select {
	case <-store.latestReady:
		store.releaseLatest()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for both concurrent verifications to read the code")
	}

	successes := 0
	invalidCodes := 0
	for i := 0; i < 2; i++ {
		select {
		case err := <-errs:
			switch {
			case err == nil:
				successes++
			case errors.Is(err, model.ErrInvalidCode):
				invalidCodes++
			default:
				t.Fatalf("unexpected verify error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for verification result")
		}
	}

	if successes != 1 || invalidCodes != 1 {
		t.Fatalf("expected one success and one invalid code error, got %d successes and %d invalid code errors", successes, invalidCodes)
	}
	if sessions := store.sessionCount(); sessions != 1 {
		t.Fatalf("expected one session, got %d", sessions)
	}
}

type replayCodeSessionStore struct {
	code          *model.EmailVerificationCode
	latestReady   chan struct{}
	latestRelease chan struct{}
	releaseOnce   sync.Once

	mu          sync.Mutex
	latestCalls int
	consumed    bool
	sessions    int
}

func newReplayCodeSessionStore(code *model.EmailVerificationCode) *replayCodeSessionStore {
	return &replayCodeSessionStore{
		code:          code,
		latestReady:   make(chan struct{}),
		latestRelease: make(chan struct{}),
	}
}

func (s *replayCodeSessionStore) releaseLatest() {
	s.releaseOnce.Do(func() {
		close(s.latestRelease)
	})
}

func (s *replayCodeSessionStore) InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error) {
	return nil, model.ErrInvalidInput
}

func (s *replayCodeSessionStore) LatestCode(ctx context.Context, email, purpose string) (*model.EmailVerificationCode, error) {
	if email != s.code.Email || purpose != s.code.Purpose {
		return nil, model.ErrNotFound
	}

	s.mu.Lock()
	s.latestCalls++
	if s.latestCalls == 2 {
		close(s.latestReady)
	}
	s.mu.Unlock()

	select {
	case <-s.latestRelease:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	code := *s.code
	return &code, nil
}

func (s *replayCodeSessionStore) IncrementCodeAttempts(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *replayCodeSessionStore) ConsumeCode(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != s.code.ID || s.consumed {
		return model.ErrInvalidCode
	}
	s.consumed = true
	return nil
}

func (s *replayCodeSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, sessionHash, userAgent, ip string, expiresAt time.Time) (*model.Session, error) {
	s.mu.Lock()
	s.sessions++
	s.mu.Unlock()

	return &model.Session{
		ID:          uuid.New(),
		UserID:      userID,
		SessionHash: sessionHash,
		UserAgent:   userAgent,
		IP:          ip,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *replayCodeSessionStore) GetSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.Session, *model.User, error) {
	return nil, nil, model.ErrNotFound
}

func (s *replayCodeSessionStore) TouchSessionLastSeen(ctx context.Context, id uuid.UUID, now, cutoff time.Time) error {
	return nil
}

func (s *replayCodeSessionStore) RevokeSession(ctx context.Context, sessionHash string) error {
	return nil
}

func (s *replayCodeSessionStore) sessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions
}
