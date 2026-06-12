package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestRequestLoginCodeVerifiesTurnstileBeforeSending(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	users := &turnstileUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com"}}
	codes := &turnstileCodeStore{}
	mailer := &turnstileMailer{}
	verifier := &recordingHumanVerifier{}
	svc := newTurnstileTestService(client, users, codes, mailer, verifier)

	if err := svc.RequestLoginCode(context.Background(), "dev@example.com", "203.0.113.10", "ts-token"); err != nil {
		t.Fatalf("RequestLoginCode returned error: %v", err)
	}
	if verifier.calls != 1 || verifier.token != "ts-token" || verifier.ip != "203.0.113.10" {
		t.Fatalf("verifier calls=%d token=%q ip=%q", verifier.calls, verifier.token, verifier.ip)
	}
	if users.getByEmailCalls != 1 {
		t.Fatalf("GetByEmail calls = %d, want 1", users.getByEmailCalls)
	}
	if codes.insertCalls != 1 || codes.insertedPurpose != "login" || codes.insertedEmail != "dev@example.com" {
		t.Fatalf("InsertCode calls=%d purpose=%q email=%q", codes.insertCalls, codes.insertedPurpose, codes.insertedEmail)
	}
	if mailer.sendCalls != 1 || mailer.sentPurpose != "login" || mailer.sentTo != "dev@example.com" {
		t.Fatalf("mailer calls=%d purpose=%q to=%q", mailer.sendCalls, mailer.sentPurpose, mailer.sentTo)
	}
}

func TestRequestLoginCodeStopsWhenTurnstileFails(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	users := &turnstileUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com"}}
	codes := &turnstileCodeStore{}
	mailer := &turnstileMailer{}
	verifier := &recordingHumanVerifier{err: model.ErrTurnstileFailed}
	svc := newTurnstileTestService(client, users, codes, mailer, verifier)

	err := svc.RequestLoginCode(context.Background(), "dev@example.com", "203.0.113.10", "bad-token")
	if !errors.Is(err, model.ErrTurnstileFailed) {
		t.Fatalf("RequestLoginCode error = %v, want ErrTurnstileFailed", err)
	}
	if verifier.calls != 1 {
		t.Fatalf("verifier calls = %d, want 1", verifier.calls)
	}
	if users.getByEmailCalls != 0 {
		t.Fatalf("GetByEmail calls = %d, want 0", users.getByEmailCalls)
	}
	if codes.insertCalls != 0 || mailer.sendCalls != 0 {
		t.Fatalf("side effects: insertCalls=%d sendCalls=%d", codes.insertCalls, mailer.sendCalls)
	}
	for _, key := range server.Keys() {
		if strings.HasPrefix(key, "email_code_cooldown:") {
			t.Fatalf("unexpected cooldown key %q", key)
		}
	}
}

func TestRequestRegisterCodeValidatesInputBeforeTurnstile(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	tests := []struct {
		name        string
		email       string
		displayName string
	}{
		{name: "invalid email", email: "not-an-email", displayName: "Dev"},
		{name: "invalid display name", email: "dev@example.com", displayName: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &recordingHumanVerifier{}
			svc := newTurnstileTestService(client, &turnstileUserStore{}, &turnstileCodeStore{}, &turnstileMailer{}, verifier)

			err := svc.RequestRegisterCode(context.Background(), tt.email, tt.displayName, "203.0.113.10", "ts-token")
			if !errors.Is(err, model.ErrInvalidInput) {
				t.Fatalf("RequestRegisterCode error = %v, want ErrInvalidInput", err)
			}
			if verifier.calls != 0 {
				t.Fatalf("verifier calls = %d, want 0", verifier.calls)
			}
		})
	}
}

func newTurnstileTestService(redisClient *redis.Client, users UserStore, codes CodeSessionStore, mailer *turnstileMailer, verifier *recordingHumanVerifier) *Service {
	return &Service{
		cfg: config.Config{
			SessionSecret:               "secret",
			EmailCodeTTLMinutes:         10,
			EmailCodeResendCooldownSecs: 60,
			EmailCodeMaxAttempts:        5,
		},
		users:         users,
		codes:         codes,
		redis:         redisClient,
		mailer:        mailer,
		humanVerifier: verifier,
		nowFunc:       time.Now,
	}
}

type recordingHumanVerifier struct {
	calls int
	token string
	ip    string
	err   error
}

func (v *recordingHumanVerifier) Verify(_ context.Context, token, ip string) error {
	v.calls++
	v.token = token
	v.ip = ip
	return v.err
}

type turnstileUserStore struct {
	user            *model.User
	getByEmailCalls int
}

func (s *turnstileUserStore) Create(context.Context, string, string) (*model.User, error) {
	return nil, model.ErrConflict
}

func (s *turnstileUserStore) GetByEmail(context.Context, string) (*model.User, error) {
	s.getByEmailCalls++
	if s.user == nil {
		return nil, model.ErrNotFound
	}
	return s.user, nil
}

func (s *turnstileUserStore) GetByID(context.Context, uuid.UUID) (*model.User, error) {
	if s.user == nil {
		return nil, model.ErrNotFound
	}
	return s.user, nil
}

func (s *turnstileUserStore) TouchLastLogin(context.Context, uuid.UUID) error {
	return nil
}

type turnstileCodeStore struct {
	disabledCodeSessionStore
	insertCalls int
}

func (s *turnstileCodeStore) InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error) {
	s.insertCalls++
	return s.disabledCodeSessionStore.InsertCode(ctx, email, purpose, codeHash, ip, expiresAt)
}

type turnstileMailer struct {
	disabledMailer
	sendCalls int
}

func (m *turnstileMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	m.sendCalls++
	return m.disabledMailer.SendVerificationCode(to, purpose, code, ttlMinutes)
}
