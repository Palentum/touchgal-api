package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestRequestLoginCodeSuppressesMissingAccount(t *testing.T) {
	existingCodes := &disabledCodeSessionStore{}
	existingMailer := &disabledMailer{}
	existingSvc := newCodeStartTestService(t, disabledUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com"}}, existingCodes, existingMailer, codeStartTestConfig())
	if err := existingSvc.RequestLoginCode(context.Background(), "dev@example.com", "203.0.113.10", "turnstile-token"); err != nil {
		t.Fatalf("existing account login code error: %v", err)
	}

	missingCodes := &disabledCodeSessionStore{}
	missingMailer := &disabledMailer{}
	missingSvc := newCodeStartTestService(t, disabledUserStore{}, missingCodes, missingMailer, codeStartTestConfig())
	if err := missingSvc.RequestLoginCode(context.Background(), "dev@example.com", "203.0.113.10", "turnstile-token"); err != nil {
		t.Fatalf("missing account login code error: %v", err)
	}

	if existingCodes.insertedPurpose != "login" || existingMailer.sentTo != "dev@example.com" {
		t.Fatalf("existing account should receive login code, purpose=%q sentTo=%q", existingCodes.insertedPurpose, existingMailer.sentTo)
	}
	if missingCodes.insertedPurpose != "" || missingMailer.sentTo != "" {
		t.Fatalf("missing account must not receive login code, purpose=%q sentTo=%q", missingCodes.insertedPurpose, missingMailer.sentTo)
	}
}

func TestRequestRegisterCodeSuppressesExistingAccount(t *testing.T) {
	existingCodes := &disabledCodeSessionStore{}
	existingMailer := &disabledMailer{}
	existingSvc := newCodeStartTestService(t, disabledUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com"}}, existingCodes, existingMailer, codeStartTestConfig())
	if err := existingSvc.RequestRegisterCode(context.Background(), "dev@example.com", "Dev", "203.0.113.10", "turnstile-token"); err != nil {
		t.Fatalf("existing account register code error: %v", err)
	}

	missingCodes := &disabledCodeSessionStore{}
	missingMailer := &disabledMailer{}
	missingSvc := newCodeStartTestService(t, disabledUserStore{}, missingCodes, missingMailer, codeStartTestConfig())
	if err := missingSvc.RequestRegisterCode(context.Background(), "dev@example.com", "Dev", "203.0.113.10", "turnstile-token"); err != nil {
		t.Fatalf("missing account register code error: %v", err)
	}

	if existingCodes.insertedPurpose != "" || existingMailer.sentTo != "" {
		t.Fatalf("existing account must not receive register code, purpose=%q sentTo=%q", existingCodes.insertedPurpose, existingMailer.sentTo)
	}
	if missingCodes.insertedPurpose != "register" || missingMailer.sentTo != "dev@example.com" {
		t.Fatalf("missing account should receive register code, purpose=%q sentTo=%q", missingCodes.insertedPurpose, missingMailer.sentTo)
	}
}

func TestRequestLoginCodeRateLimitsEmailAndIPEmail(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.Config
		firstIP  string
		secondIP string
	}{
		{
			name:     "email",
			cfg:      withCodeStartRateLimits(100, 1, 100, 100),
			firstIP:  "203.0.113.10",
			secondIP: "203.0.113.11",
		},
		{
			name:     "ip email",
			cfg:      withCodeStartRateLimits(100, 100, 100, 1),
			firstIP:  "203.0.113.10",
			secondIP: "203.0.113.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: server.Addr()})
			defer client.Close()

			svc := &Service{
				cfg:           tt.cfg,
				users:         disabledUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com"}},
				codes:         &disabledCodeSessionStore{},
				redis:         client,
				mailer:        &disabledMailer{},
				humanVerifier: nil,
				nowFunc:       time.Now,
			}

			if err := svc.RequestLoginCode(context.Background(), "dev@example.com", tt.firstIP, "turnstile-token"); err != nil {
				t.Fatalf("first request error: %v", err)
			}
			server.FastForward(2 * time.Second)
			err := svc.RequestLoginCode(context.Background(), "dev@example.com", tt.secondIP, "turnstile-token")
			if !errors.Is(err, model.ErrRateLimited) {
				t.Fatalf("second request error = %v, want ErrRateLimited", err)
			}
		})
	}
}

func TestRequestLoginCodeDoesNotSetCooldownOnLookupError(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	lookupErr := errors.New("lookup failed")
	svc := &Service{
		cfg:           codeStartTestConfig(),
		users:         errorUserStore{err: lookupErr},
		codes:         &disabledCodeSessionStore{},
		redis:         client,
		mailer:        &disabledMailer{},
		humanVerifier: nil,
		nowFunc:       time.Now,
	}

	err := svc.RequestLoginCode(context.Background(), "dev@example.com", "203.0.113.10", "turnstile-token")
	if !errors.Is(err, lookupErr) {
		t.Fatalf("RequestLoginCode error = %v, want lookup error", err)
	}
	if server.Exists("email_code_cooldown:login:dev@example.com") {
		t.Fatal("lookup failure must not set login cooldown")
	}
}

func newCodeStartTestService(t *testing.T, users UserStore, codes CodeSessionStore, mailer *disabledMailer, cfg config.Config) *Service {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &Service{
		cfg:           cfg,
		users:         users,
		codes:         codes,
		redis:         client,
		mailer:        mailer,
		humanVerifier: nil,
		nowFunc:       time.Now,
	}
}

func codeStartTestConfig() config.Config {
	return withCodeStartRateLimits(100, 100, 100, 100)
}

func withCodeStartRateLimits(emailMinute, emailDaily, ipEmailMinute, ipEmailDaily int) config.Config {
	return config.Config{
		SessionSecret:               "secret",
		EmailCodeTTLMinutes:         10,
		EmailCodeResendCooldownSecs: 1,
		EmailCodeMaxAttempts:        5,
		AuthCodeEmailMinuteLimit:    emailMinute,
		AuthCodeEmailDailyLimit:     emailDaily,
		AuthCodeIPEmailMinuteLimit:  ipEmailMinute,
		AuthCodeIPEmailDailyLimit:   ipEmailDaily,
	}
}

type errorUserStore struct {
	err error
}

func (s errorUserStore) Create(context.Context, string, string) (*model.User, error) {
	return nil, s.err
}

func (s errorUserStore) GetByEmail(context.Context, string) (*model.User, error) {
	return nil, s.err
}

func (s errorUserStore) GetByID(context.Context, uuid.UUID) (*model.User, error) {
	return nil, s.err
}

func (s errorUserStore) TouchLastLogin(context.Context, uuid.UUID) error {
	return s.err
}
