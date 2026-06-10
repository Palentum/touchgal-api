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

func TestRequestLoginCodeAllowsDisabledUser(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	codes := &disabledCodeSessionStore{}
	mailer := &disabledMailer{}
	svc := &Service{
		cfg: config.Config{
			SessionSecret:               "secret",
			EmailCodeTTLMinutes:         10,
			EmailCodeResendCooldownSecs: 60,
			EmailCodeMaxAttempts:        5,
		},
		users:   disabledUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com", Status: model.UserStatusDisabled}},
		codes:   codes,
		redis:   client,
		mailer:  mailer,
		nowFunc: time.Now,
	}

	if err := svc.RequestLoginCode(context.Background(), "dev@example.com", "127.0.0.1"); err != nil {
		t.Fatalf("expected disabled user to receive login code, got %v", err)
	}
	if codes.insertedPurpose != "login" || codes.insertedEmail != "dev@example.com" {
		t.Fatalf("expected login code insertion, got purpose=%q email=%q", codes.insertedPurpose, codes.insertedEmail)
	}
	if mailer.sentTo != "dev@example.com" || mailer.sentPurpose != "login" {
		t.Fatalf("expected login mail, got purpose=%q email=%q", mailer.sentPurpose, mailer.sentTo)
	}
}

func TestCreateSessionAllowsDisabledUser(t *testing.T) {
	codes := &disabledCodeSessionStore{}
	userID := uuid.New()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	svc := &Service{
		cfg:     config.Config{SessionSecret: "secret", SessionTTLHours: 24},
		users:   disabledUserStore{user: &model.User{ID: userID, Status: model.UserStatusDisabled}},
		codes:   codes,
		nowFunc: func() time.Time { return now },
	}

	result, err := svc.createSession(context.Background(), &model.User{ID: userID, Status: model.UserStatusDisabled}, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("expected disabled user session, got %v", err)
	}
	if result.SessionToken == "" {
		t.Fatal("expected plaintext session token")
	}
	if result.User.Status != model.UserStatusDisabled {
		t.Fatalf("expected disabled user in result, got %q", result.User.Status)
	}
	if codes.sessionUserID != userID {
		t.Fatalf("expected session for disabled user %s, got %s", userID, codes.sessionUserID)
	}
	if !codes.sessionExpiresAt.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("unexpected expiry %s", codes.sessionExpiresAt)
	}
}

type disabledUserStore struct {
	user *model.User
}

func (s disabledUserStore) Create(ctx context.Context, email, displayName string) (*model.User, error) {
	return nil, model.ErrConflict
}

func (s disabledUserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if s.user == nil {
		return nil, model.ErrNotFound
	}
	return s.user, nil
}

func (s disabledUserStore) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if s.user == nil {
		return nil, model.ErrNotFound
	}
	return s.user, nil
}

func (s disabledUserStore) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	return nil
}

type disabledCodeSessionStore struct {
	insertedEmail    string
	insertedPurpose  string
	sessionUserID    uuid.UUID
	sessionExpiresAt time.Time
}

func (s *disabledCodeSessionStore) InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error) {
	s.insertedEmail = email
	s.insertedPurpose = purpose
	return &model.EmailVerificationCode{ID: uuid.New(), Email: email, Purpose: purpose, CodeHash: codeHash, IP: ip, ExpiresAt: expiresAt}, nil
}

func (s *disabledCodeSessionStore) LatestCode(ctx context.Context, email, purpose string) (*model.EmailVerificationCode, error) {
	return nil, model.ErrNotFound
}

func (s *disabledCodeSessionStore) IncrementCodeAttempts(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *disabledCodeSessionStore) ConsumeCode(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *disabledCodeSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, sessionHash, userAgent, ip string, expiresAt time.Time) (*model.Session, error) {
	s.sessionUserID = userID
	s.sessionExpiresAt = expiresAt
	return &model.Session{ID: uuid.New(), UserID: userID, SessionHash: sessionHash, UserAgent: userAgent, IP: ip, ExpiresAt: expiresAt}, nil
}

func (s *disabledCodeSessionStore) GetSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.Session, *model.User, error) {
	return nil, nil, model.ErrNotFound
}

func (s *disabledCodeSessionStore) TouchSessionLastSeen(ctx context.Context, id uuid.UUID, now, cutoff time.Time) error {
	return nil
}

func (s *disabledCodeSessionStore) RevokeSession(ctx context.Context, sessionHash string) error {
	return nil
}

type disabledMailer struct {
	sentTo      string
	sentPurpose string
}

func (m *disabledMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	m.sentTo = to
	m.sentPurpose = purpose
	return nil
}
