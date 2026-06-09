package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/services/email"
)

type UserStore interface {
	Create(ctx context.Context, email, displayName string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	TouchLastLogin(ctx context.Context, id uuid.UUID) error
}

type CodeSessionStore interface {
	InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error)
	LatestCode(ctx context.Context, email, purpose string) (*model.EmailVerificationCode, error)
	IncrementCodeAttempts(ctx context.Context, id uuid.UUID) error
	ConsumeCode(ctx context.Context, id uuid.UUID) error
	CreateSession(ctx context.Context, userID uuid.UUID, sessionHash, userAgent, ip string, expiresAt time.Time) (*model.Session, error)
	GetSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.Session, *model.User, error)
	RevokeSession(ctx context.Context, sessionHash string) error
}

type Service struct {
	cfg     config.Config
	users   UserStore
	codes   CodeSessionStore
	redis   *redis.Client
	mailer  email.Mailer
	nowFunc func() time.Time
}

type VerifyResult struct {
	User         *model.User
	SessionToken string
	ExpiresAt    time.Time
}

func NewService(cfg config.Config, users UserStore, codes CodeSessionStore, redisClient *redis.Client, mailer email.Mailer) *Service {
	return &Service{cfg: cfg, users: users, codes: codes, redis: redisClient, mailer: mailer, nowFunc: time.Now}
}

func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > 254 {
		return "", model.ErrInvalidInput
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", model.ErrInvalidInput
	}
	return email, nil
}

func (s *Service) RequestRegisterCode(ctx context.Context, rawEmail, displayName, ip string) error {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 80 {
		return model.ErrInvalidInput
	}
	if _, err := s.users.GetByEmail(ctx, emailAddr); err == nil {
		return model.ErrConflict
	} else if !errors.Is(err, model.ErrNotFound) {
		return err
	}
	if err := s.setCooldown(ctx, "register", emailAddr); err != nil {
		return err
	}
	if err := s.redis.Set(ctx, "email_code_display:register:"+emailAddr, displayName, s.cfg.EmailCodeTTL()).Err(); err != nil {
		return err
	}
	return s.issueCode(ctx, "register", emailAddr, ip)
}

func (s *Service) RequestLoginCode(ctx context.Context, rawEmail, ip string) error {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	if _, err := s.users.GetByEmail(ctx, emailAddr); err != nil {
		return err
	}
	if err := s.setCooldown(ctx, "login", emailAddr); err != nil {
		return err
	}
	return s.issueCode(ctx, "login", emailAddr, ip)
}

func (s *Service) VerifyRegister(ctx context.Context, rawEmail, code, userAgent, ip string) (*VerifyResult, error) {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return nil, err
	}
	if err := s.verifyCode(ctx, "register", emailAddr, code); err != nil {
		return nil, err
	}
	displayName, err := s.redis.Get(ctx, "email_code_display:register:"+emailAddr).Result()
	if err != nil {
		displayName = strings.Split(emailAddr, "@")[0]
	}
	user, err := s.users.Create(ctx, emailAddr, displayName)
	if err != nil {
		return nil, err
	}
	return s.createSession(ctx, user, userAgent, ip)
}

func (s *Service) VerifyLogin(ctx context.Context, rawEmail, code, userAgent, ip string) (*VerifyResult, error) {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return nil, err
	}
	if err := s.verifyCode(ctx, "login", emailAddr, code); err != nil {
		return nil, err
	}
	user, err := s.users.GetByEmail(ctx, emailAddr)
	if err != nil {
		return nil, err
	}
	return s.createSession(ctx, user, userAgent, ip)
}

func (s *Service) CurrentUser(ctx context.Context, rawSessionToken string) (*model.User, error) {
	if rawSessionToken == "" {
		return nil, model.ErrUnauthorized
	}
	hash := HashSessionToken(rawSessionToken, s.cfg.SessionSecret)
	_, user, err := s.codes.GetSessionUser(ctx, hash, s.nowFunc())
	if err != nil {
		return nil, model.ErrUnauthorized
	}
	return user, nil
}

func (s *Service) RevokeSession(ctx context.Context, rawSessionToken string) error {
	if rawSessionToken == "" {
		return nil
	}
	return s.codes.RevokeSession(ctx, HashSessionToken(rawSessionToken, s.cfg.SessionSecret))
}

func (s *Service) issueCode(ctx context.Context, purpose, emailAddr, ip string) error {
	code, err := GenerateNumericCode()
	if err != nil {
		return err
	}
	hash := HashCode(purpose, emailAddr, code, s.cfg.SessionSecret)
	if _, err := s.codes.InsertCode(ctx, emailAddr, purpose, hash, ip, s.nowFunc().Add(s.cfg.EmailCodeTTL())); err != nil {
		return err
	}
	if err := s.redis.Set(ctx, "email_code_hash:"+purpose+":"+emailAddr, hash, s.cfg.EmailCodeTTL()).Err(); err != nil {
		return err
	}
	return s.mailer.SendVerificationCode(emailAddr, purpose, code, s.cfg.EmailCodeTTLMinutes)
}

func (s *Service) setCooldown(ctx context.Context, purpose, emailAddr string) error {
	ok, err := s.redis.SetNX(ctx, "email_code_cooldown:"+purpose+":"+emailAddr, "1", s.cfg.EmailCooldown()).Result()
	if err != nil {
		return err
	}
	if !ok {
		return model.ErrCodeCooldown
	}
	return nil
}

func (s *Service) verifyCode(ctx context.Context, purpose, emailAddr, code string) error {
	if len(code) != 6 {
		return model.ErrInvalidCode
	}
	record, err := s.codes.LatestCode(ctx, emailAddr, purpose)
	if err != nil {
		return model.ErrInvalidCode
	}
	if record.ConsumedAt != nil {
		return model.ErrInvalidCode
	}
	if s.nowFunc().After(record.ExpiresAt) {
		return model.ErrExpiredCode
	}
	if record.Attempts >= s.cfg.EmailCodeMaxAttempts {
		return model.ErrInvalidCode
	}
	if !VerifyCodeHash(purpose, emailAddr, code, s.cfg.SessionSecret, record.CodeHash) {
		_ = s.codes.IncrementCodeAttempts(ctx, record.ID)
		return model.ErrInvalidCode
	}
	return s.codes.ConsumeCode(ctx, record.ID)
}

func (s *Service) createSession(ctx context.Context, user *model.User, userAgent, ip string) (*VerifyResult, error) {
	raw, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	expiresAt := s.nowFunc().Add(s.cfg.SessionTTL())
	if _, err := s.codes.CreateSession(ctx, user.ID, HashSessionToken(raw, s.cfg.SessionSecret), userAgent, ip, expiresAt); err != nil {
		return nil, err
	}
	_ = s.users.TouchLastLogin(ctx, user.ID)
	return &VerifyResult{User: user, SessionToken: raw, ExpiresAt: expiresAt}, nil
}
