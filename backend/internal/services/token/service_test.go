package token

import (
	"context"
	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"testing"
	"time"
)

type fakeTokenStore struct {
	created          *model.APIToken
	auth             *model.TokenAuthInfo
	deletedForUserID uuid.UUID
	deletedUserID    uuid.UUID
	deletedID        uuid.UUID
	updatedName      string
	updatedForUserID uuid.UUID
	updatedUserID    uuid.UUID
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
func (f *fakeTokenStore) UpdateNameForUser(ctx context.Context, id, userID uuid.UUID, name string) (*model.APIToken, error) {
	f.updatedForUserID = id
	f.updatedUserID = userID
	f.updatedName = name
	return &model.APIToken{ID: id, UserID: userID, Name: name, Status: model.TokenActive}, nil
}
func (f *fakeTokenStore) DeleteForUser(ctx context.Context, id, userID uuid.UUID) error {
	f.deletedForUserID = id
	f.deletedUserID = userID
	return nil
}
func (f *fakeTokenStore) DeleteByID(ctx context.Context, id uuid.UUID) error {
	f.deletedID = id
	return nil
}
func (f *fakeTokenStore) UpdateLastUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeTokenStore) CountActive(ctx context.Context) (int, error)           { return 0, nil }

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

func TestUpdateNameMineTrimsAndScopesToken(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{}
	svc := NewService(config.Config{}, store, fakeTokenAppStore{})

	updated, err := svc.UpdateNameMine(context.Background(), tokenID, userID, "  renamed token  ")
	if err != nil {
		t.Fatalf("update token name: %v", err)
	}
	if updated.Name != "renamed token" || store.updatedName != "renamed token" {
		t.Fatal("token name must be trimmed before persistence")
	}
	if store.updatedForUserID != tokenID || store.updatedUserID != userID {
		t.Fatal("token rename must be scoped to the current user")
	}
}

func TestUpdateNameMineRejectsBlankName(t *testing.T) {
	svc := NewService(config.Config{}, &fakeTokenStore{}, fakeTokenAppStore{})
	if _, err := svc.UpdateNameMine(context.Background(), uuid.New(), uuid.New(), "   "); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid input for blank token name, got %v", err)
	}
}

func TestDeleteMineRemovesTokenRecord(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{}
	svc := NewService(config.Config{}, store, fakeTokenAppStore{})

	if err := svc.DeleteMine(context.Background(), tokenID, userID); err != nil {
		t.Fatalf("delete token: %v", err)
	}
	if store.deletedForUserID != tokenID || store.deletedUserID != userID {
		t.Fatal("delete must remove the token row scoped to the current user")
	}
}

func TestAuthenticateRejectsDeletedToken(t *testing.T) {
	cfg := config.Config{APITokenPepper: "pepper"}
	raw := "tgal_live_test"
	store := &fakeTokenStore{}
	svc := NewService(cfg, store, fakeTokenAppStore{})
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected deleted token unauthorized, got %v", err)
	}
}

func TestAuthenticateDeletesInactiveTokenRecord(t *testing.T) {
	cfg := config.Config{APITokenPepper: "pepper"}
	raw := "tgal_live_test"
	tokenID := uuid.New()
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{Token: model.APIToken{ID: tokenID, Status: "disabled", TokenHash: HashAPIToken(raw, "pepper")}, ApplicationStatus: model.ApplicationApproved}}
	svc := NewService(cfg, store, fakeTokenAppStore{})
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected inactive token unauthorized, got %v", err)
	}
	if store.deletedID != tokenID {
		t.Fatal("inactive token record must be deleted instead of kept as disabled/revoked")
	}
}

func TestAuthenticateDeletesExpiredTokenRecord(t *testing.T) {
	cfg := config.Config{APITokenPepper: "pepper"}
	raw := "tgal_live_test"
	tokenID := uuid.New()
	expiredAt := time.Now().Add(-time.Second)
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{Token: model.APIToken{ID: tokenID, Status: model.TokenActive, TokenHash: HashAPIToken(raw, "pepper"), ExpiresAt: &expiredAt}, ApplicationStatus: model.ApplicationApproved}}
	svc := NewService(cfg, store, fakeTokenAppStore{})
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected expired token unauthorized, got %v", err)
	}
	if store.deletedID != tokenID {
		t.Fatal("expired token record must be deleted instead of kept in the database")
	}
}
