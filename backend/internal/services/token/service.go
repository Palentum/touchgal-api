package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type Store interface {
	Create(ctx context.Context, token model.APIToken) (*model.APIToken, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error)
	ListAdmin(ctx context.Context, status string, page, limit int) ([]model.APIToken, error)
	GetByHashWithApplication(ctx context.Context, tokenHash string) (*model.TokenAuthInfo, error)
	RevokeForUser(ctx context.Context, id, userID uuid.UUID) error
	RevokeByAdmin(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	CountActive(ctx context.Context) (int, error)
}

type ApplicationStore interface {
	GetApprovedByUser(ctx context.Context, userID uuid.UUID) (*model.Application, error)
}

type Service struct {
	cfg          config.Config
	tokens       Store
	applications ApplicationStore
}

type CreateResult struct {
	Token      *model.APIToken `json:"tokenRecord"`
	PlainToken string          `json:"token"`
}

func NewService(cfg config.Config, tokens Store, applications ApplicationStore) *Service {
	return &Service{cfg: cfg, tokens: tokens, applications: applications}
}

func GenerateAPIToken(prefix string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashAPIToken(token, pepper string) string {
	sum := sha256.Sum256([]byte(token + "." + pepper))
	return hex.EncodeToString(sum[:])
}

func ConstantTimeTokenHashEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func TokenPrefix(raw string) string {
	if len(raw) <= 32 {
		return raw
	}
	return raw[:32]
}

func CanCreateFromApplication(app *model.Application) bool {
	return app != nil && app.Status == model.ApplicationApproved
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name string) (*CreateResult, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return nil, model.ErrInvalidInput
	}
	app, err := s.applications.GetApprovedByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.ErrApplicationOpen
		}
		return nil, err
	}
	if !CanCreateFromApplication(app) {
		return nil, model.ErrApplicationOpen
	}
	raw, err := GenerateAPIToken(s.cfg.APITokenPrefix)
	if err != nil {
		return nil, err
	}
	record := model.APIToken{
		ID: uuid.New(), UserID: userID, ApplicationID: app.ID, Name: name,
		TokenPrefix: TokenPrefix(raw), TokenHash: HashAPIToken(raw, s.cfg.APITokenPepper),
		MinuteLimit: app.DefaultMinuteLimit, DailyLimit: app.DefaultDailyLimit,
	}
	if record.MinuteLimit <= 0 {
		record.MinuteLimit = s.cfg.DefaultTokenMinuteLimit
	}
	if record.DailyLimit <= 0 {
		record.DailyLimit = s.cfg.DefaultTokenDailyLimit
	}
	created, err := s.tokens.Create(ctx, record)
	if err != nil {
		return nil, err
	}
	return &CreateResult{Token: created, PlainToken: raw}, nil
}

func (s *Service) ListMine(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return s.tokens.ListByUser(ctx, userID)
}
func (s *Service) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.APIToken, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.tokens.ListAdmin(ctx, status, page, limit)
}
func (s *Service) RevokeMine(ctx context.Context, id, userID uuid.UUID) error {
	return s.tokens.RevokeForUser(ctx, id, userID)
}
func (s *Service) RevokeByAdmin(ctx context.Context, id uuid.UUID) error {
	return s.tokens.RevokeByAdmin(ctx, id)
}
func (s *Service) CountActive(ctx context.Context) (int, error) { return s.tokens.CountActive(ctx) }

func (s *Service) Authenticate(ctx context.Context, raw string) (*model.TokenAuthInfo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, model.ErrUnauthorized
	}
	info, err := s.tokens.GetByHashWithApplication(ctx, HashAPIToken(raw, s.cfg.APITokenPepper))
	if err != nil {
		return nil, model.ErrUnauthorized
	}
	if info.Token.Status != model.TokenActive || info.ApplicationStatus != model.ApplicationApproved {
		return nil, model.ErrUnauthorized
	}
	if info.Token.ExpiresAt != nil && time.Now().After(*info.Token.ExpiresAt) {
		return nil, model.ErrUnauthorized
	}
	_ = s.tokens.UpdateLastUsed(ctx, info.Token.ID)
	return info, nil
}
