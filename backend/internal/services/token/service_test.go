package token

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeTokenStore struct {
	created *model.APIToken
	auth    *model.TokenAuthInfo
}

func (f *fakeTokenStore) Create(ctx context.Context, token model.APIToken) (*model.APIToken, error) {
	token.Status = model.TokenActive
	f.created = &token
	return &token, nil
}
func (f *fakeTokenStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return nil, nil
}
func (f *fakeTokenStore) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.APIToken, error) {
	return nil, nil
}
func (f *fakeTokenStore) GetByHashWithApplication(ctx context.Context, tokenHash string) (*model.TokenAuthInfo, error) {
	if f.auth == nil {
		return nil, model.ErrNotFound
	}
	return f.auth, nil
}
func (f *fakeTokenStore) RevokeForUser(ctx context.Context, id, userID uuid.UUID) error { return nil }
func (f *fakeTokenStore) RevokeByAdmin(ctx context.Context, id uuid.UUID) error         { return nil }
func (f *fakeTokenStore) UpdateLastUsed(ctx context.Context, id uuid.UUID) error        { return nil }
func (f *fakeTokenStore) CountActive(ctx context.Context) (int, error)                  { return 0, nil }

type fakeTokenAppStore struct {
	app          *model.Application
	ensuredAdmin *model.Application
}

func (f fakeTokenAppStore) GetApprovedByUser(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	if f.app == nil {
		return nil, model.ErrNotFound
	}
	return f.app, nil
}

func (f fakeTokenAppStore) EnsureAdminApproved(ctx context.Context, userID uuid.UUID, minuteLimit, dailyLimit int) (*model.Application, error) {
	if f.ensuredAdmin == nil {
		return nil, model.ErrNotFound
	}
	f.ensuredAdmin.UserID = userID
	f.ensuredAdmin.DefaultMinuteLimit = minuteLimit
	f.ensuredAdmin.DefaultDailyLimit = dailyLimit
	return f.ensuredAdmin, nil
}

func TestTokenOnlyApprovedAccountCanCreate(t *testing.T) {
	cfg := config.Config{APITokenPrefix: "tgal_live", APITokenPepper: "pepper", DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}
	pending := &model.Application{ID: uuid.New(), UserID: uuid.New(), Status: model.ApplicationPending}
	svc := NewService(cfg, &fakeTokenStore{}, fakeTokenAppStore{app: pending})
	_, err := svc.Create(context.Background(), pending.UserID, false, "prod")
	if err != model.ErrApplicationOpen {
		t.Fatalf("expected application error, got %v", err)
	}

	approved := &model.Application{ID: uuid.New(), UserID: uuid.New(), Status: model.ApplicationApproved, DefaultMinuteLimit: 10, DefaultDailyLimit: 100}
	store := &fakeTokenStore{}
	svc = NewService(cfg, store, fakeTokenAppStore{app: approved})
	res, err := svc.Create(context.Background(), approved.UserID, false, "prod")
	if err != nil {
		t.Fatalf("create approved token: %v", err)
	}
	if res.PlainToken == "" || store.created.TokenHash == res.PlainToken {
		t.Fatal("plaintext must be returned once and hash stored separately")
	}
	if store.created.TokenPrefix == "" || len(store.created.TokenPrefix) > 32 {
		t.Fatal("invalid token prefix")
	}
	if store.created.ApplicationID != approved.ID {
		t.Fatal("token must be tied to the account application selected by the service")
	}
}

func TestAdminCanCreateTokenWithoutSubmittedApplication(t *testing.T) {
	cfg := config.Config{APITokenPrefix: "tgal_live", APITokenPepper: "pepper", DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}
	userID := uuid.New()
	adminApp := &model.Application{ID: uuid.New(), Status: model.ApplicationApproved}
	store := &fakeTokenStore{}
	svc := NewService(cfg, store, fakeTokenAppStore{ensuredAdmin: adminApp})

	res, err := svc.Create(context.Background(), userID, true, "prod")
	if err != nil {
		t.Fatalf("admin create token: %v", err)
	}
	if store.created.ApplicationID != adminApp.ID {
		t.Fatal("admin token must be tied to the default approved application")
	}
	if res.Token.MinuteLimit != cfg.DefaultTokenMinuteLimit || res.Token.DailyLimit != cfg.DefaultTokenDailyLimit {
		t.Fatal("admin default application limits must be applied to the token")
	}
}

func TestAuthenticateRejectsRevokedToken(t *testing.T) {
	cfg := config.Config{APITokenPepper: "pepper"}
	raw := "tgal_live_test"
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{Token: model.APIToken{ID: uuid.New(), Status: model.TokenRevoked, TokenHash: HashAPIToken(raw, "pepper")}, ApplicationStatus: model.ApplicationApproved}}
	svc := NewService(cfg, store, fakeTokenAppStore{})
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected revoked token unauthorized, got %v", err)
	}
}
