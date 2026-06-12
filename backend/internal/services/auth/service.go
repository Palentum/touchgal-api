package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/services/email"
	"github.com/touchgal/developer/backend/internal/services/turnstile"
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
	TouchSessionLastSeen(ctx context.Context, id uuid.UUID, now, cutoff time.Time) error
	RevokeSession(ctx context.Context, sessionHash string) error
}

type Service struct {
	cfg           config.Config
	users         UserStore
	codes         CodeSessionStore
	redis         *redis.Client
	mailer        email.Mailer
	humanVerifier turnstile.Verifier
	nowFunc       func() time.Time
}

type VerifyResult struct {
	User         *model.User
	SessionToken string
	ExpiresAt    time.Time
}

type cachedSessionUser struct {
	SessionID uuid.UUID  `json:"sessionId"`
	User      model.User `json:"user"`
	ExpiresAt time.Time  `json:"expiresAt"`
}

func NewService(cfg config.Config, users UserStore, codes CodeSessionStore, redisClient *redis.Client, mailer email.Mailer) *Service {
	return &Service{
		cfg:           cfg,
		users:         users,
		codes:         codes,
		redis:         redisClient,
		mailer:        mailer,
		humanVerifier: turnstile.New(cfg.TurnstileSecretKey),
		nowFunc:       time.Now,
	}
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

func (s *Service) verifyHuman(ctx context.Context, token, ip string) error {
	if s.humanVerifier == nil {
		return nil
	}
	return s.humanVerifier.Verify(ctx, token, ip)
}

func (s *Service) RequestRegisterCode(ctx context.Context, rawEmail, displayName, ip, turnstileToken string) error {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 80 {
		return model.ErrInvalidInput
	}
	if err := s.verifyHuman(ctx, turnstileToken, ip); err != nil {
		return err
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

func (s *Service) RequestLoginCode(ctx context.Context, rawEmail, ip, turnstileToken string) error {
	emailAddr, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	if err := s.verifyHuman(ctx, turnstileToken, ip); err != nil {
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
	now := s.nowFunc()
	hash := HashSessionToken(rawSessionToken, s.cfg.SessionSecret)
	if user, sessionID, ok := s.cachedSessionUser(ctx, hash, now); ok {
		s.touchSessionLastSeen(ctx, sessionID, now)
		return user, nil
	}
	session, user, err := s.codes.GetSessionUser(ctx, hash, now)
	if err != nil {
		s.deleteSessionCache(ctx, hash)
		return nil, model.ErrUnauthorized
	}
	s.cacheSessionUser(ctx, hash, session, user, now)
	s.touchSessionLastSeen(ctx, session.ID, now)
	return user, nil
}

func (s *Service) RevokeSession(ctx context.Context, rawSessionToken string) error {
	if rawSessionToken == "" {
		return nil
	}
	hash := HashSessionToken(rawSessionToken, s.cfg.SessionSecret)
	err := s.codes.RevokeSession(ctx, hash)
	cacheErr := s.revokeSessionCache(ctx, hash)
	if err != nil {
		return err
	}
	return cacheErr
}

func (s *Service) InvalidateUserSessions(ctx context.Context, userID uuid.UUID) {
	if s.redis == nil || userID == uuid.Nil {
		return
	}
	userKey := sessionUserSessionsKey(userID)
	keys, err := s.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return
	}
	if len(keys) == 0 {
		_ = s.redis.Del(ctx, userKey).Err()
		return
	}
	keys = append(keys, userKey)
	_ = s.redis.Del(ctx, keys...).Err()
}

func (s *Service) cachedSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.User, uuid.UUID, bool) {
	if s.redis == nil {
		return nil, uuid.Nil, false
	}
	key := sessionAuthCacheKey(sessionHash)
	values, err := s.redis.MGet(ctx, key, sessionRevokedKey(sessionHash)).Result()
	if err != nil || len(values) != 2 {
		return nil, uuid.Nil, false
	}
	if values[1] != nil {
		_ = s.redis.Del(ctx, key).Err()
		return nil, uuid.Nil, false
	}
	data, ok := values[0].(string)
	if !ok || data == "" {
		return nil, uuid.Nil, false
	}
	var cached cachedSessionUser
	if err := json.Unmarshal([]byte(data), &cached); err != nil || cached.SessionID == uuid.Nil || !now.Before(cached.ExpiresAt) {
		_ = s.redis.Del(ctx, key).Err()
		return nil, uuid.Nil, false
	}
	user := cached.User
	return &user, cached.SessionID, true
}

func (s *Service) cacheSessionUser(ctx context.Context, sessionHash string, session *model.Session, user *model.User, now time.Time) {
	if s.redis == nil || session == nil || user == nil {
		return
	}
	ttl := s.sessionAuthCacheTTL()
	untilExpiry := session.ExpiresAt.Sub(now)
	if untilExpiry <= 0 {
		return
	}
	if untilExpiry < ttl {
		ttl = untilExpiry
	}
	payload, err := json.Marshal(cachedSessionUser{SessionID: session.ID, User: *user, ExpiresAt: session.ExpiresAt})
	if err != nil {
		return
	}
	key := sessionAuthCacheKey(sessionHash)
	if err := s.redis.Set(ctx, key, payload, ttl).Err(); err != nil {
		return
	}
	userKey := sessionUserSessionsKey(user.ID)
	pipe := s.redis.Pipeline()
	pipe.SAdd(ctx, userKey, key)
	pipe.Expire(ctx, userKey, s.sessionAuthCacheTTL())
	if _, err := pipe.Exec(ctx); err != nil {
		_ = s.redis.Del(ctx, key).Err()
	}
}

func (s *Service) revokeSessionCache(ctx context.Context, sessionHash string) error {
	if s.redis == nil || sessionHash == "" {
		return nil
	}
	key := sessionAuthCacheKey(sessionHash)
	data, getErr := s.redis.Get(ctx, key).Bytes()
	pipe := s.redis.Pipeline()
	pipe.Set(ctx, sessionRevokedKey(sessionHash), "1", s.sessionAuthCacheTTL())
	pipe.Del(ctx, key)
	if getErr == nil {
		var cached cachedSessionUser
		if json.Unmarshal(data, &cached) == nil && cached.User.ID != uuid.Nil {
			pipe.SRem(ctx, sessionUserSessionsKey(cached.User.ID), key)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Service) deleteSessionCache(ctx context.Context, sessionHash string) error {
	if s.redis == nil || sessionHash == "" {
		return nil
	}
	key := sessionAuthCacheKey(sessionHash)
	data, getErr := s.redis.Get(ctx, key).Bytes()
	pipe := s.redis.Pipeline()
	pipe.Del(ctx, key)
	if getErr == nil {
		var cached cachedSessionUser
		if json.Unmarshal(data, &cached) == nil && cached.User.ID != uuid.Nil {
			pipe.SRem(ctx, sessionUserSessionsKey(cached.User.ID), key)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Service) touchSessionLastSeen(ctx context.Context, sessionID uuid.UUID, now time.Time) {
	if s.redis == nil || sessionID == uuid.Nil {
		return
	}
	interval := s.sessionLastSeenUpdateInterval()
	ok, err := s.redis.SetNX(ctx, sessionLastSeenKey(sessionID), "1", interval).Result()
	if err != nil || !ok {
		return
	}
	_ = s.codes.TouchSessionLastSeen(ctx, sessionID, now, now.Add(-interval))
}

func (s *Service) sessionAuthCacheTTL() time.Duration {
	ttl := s.cfg.SessionAuthCacheTTL()
	if ttl <= 0 {
		return time.Minute
	}
	return ttl
}

func (s *Service) sessionLastSeenUpdateInterval() time.Duration {
	interval := s.cfg.SessionLastSeenUpdateInterval()
	if interval <= 0 {
		return 5 * time.Minute
	}
	return interval
}

func sessionAuthCacheKey(sessionHash string) string {
	return "session:auth:" + sessionHash
}

func sessionRevokedKey(sessionHash string) string {
	return "session:revoked:" + sessionHash
}

func sessionUserSessionsKey(userID uuid.UUID) string {
	return "session:auth:user:" + userID.String()
}

func sessionLastSeenKey(sessionID uuid.UUID) string {
	return "session:last_seen:" + sessionID.String()
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
