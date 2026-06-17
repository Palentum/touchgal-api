package token

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeTokenStore struct {
	created               *model.APIToken
	createMaxActiveTokens int
	activeTokenCount      int
	auth                  *model.TokenAuthInfo
	authLookups           int
	deletedForUserID      uuid.UUID
	deletedUserID         uuid.UUID
	deletedID             uuid.UUID
	updatedName           string
	updatedForUserID      uuid.UUID
	updatedUserID         uuid.UUID
	lastUsedUpdates       int
	listStatus            string
	listPage              int
	listLimit             int
}

func (f *fakeTokenStore) Create(ctx context.Context, token model.APIToken, maxActiveTokensPerUser int) (*model.APIToken, error) {
	f.createMaxActiveTokens = maxActiveTokensPerUser
	if f.activeTokenCount >= maxActiveTokensPerUser {
		return nil, model.ErrTokenLimitExceeded
	}
	token.Status = model.TokenActive
	f.created = &token
	return &token, nil
}
func (f *fakeTokenStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return nil, nil
}
func (f *fakeTokenStore) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminAPIToken, error) {
	f.listStatus = status
	f.listPage = page
	f.listLimit = limit
	return nil, nil
}
func (f *fakeTokenStore) GetByHashWithApplication(ctx context.Context, tokenHash string) (*model.TokenAuthInfo, error) {
	f.authLookups++
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
func (f *fakeTokenStore) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	f.lastUsedUpdates++
	return nil
}
func (f *fakeTokenStore) CountActive(ctx context.Context) (int, error) { return 0, nil }

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
	svc := NewService(cfg, &fakeTokenStore{}, fakeTokenAppStore{app: pending}, nil)
	_, err := svc.Create(context.Background(), pending.UserID, false, "prod")
	if err != model.ErrApplicationOpen {
		t.Fatalf("expected application error, got %v", err)
	}

	approved := &model.Application{ID: uuid.New(), UserID: uuid.New(), Status: model.ApplicationApproved, DefaultMinuteLimit: 10, DefaultDailyLimit: 100}
	store := &fakeTokenStore{}
	svc = NewService(cfg, store, fakeTokenAppStore{app: approved}, nil)
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
func TestCreateTokenEnforcesActiveTokenLimit(t *testing.T) {
	userID := uuid.New()
	cfg := config.Config{
		APITokenPrefix:          "tgal_live",
		APITokenPepper:          "pepper",
		DefaultTokenMinuteLimit: 60,
		DefaultTokenDailyLimit:  5000,
		MaxActiveTokensPerUser:  2,
	}
	app := &model.Application{ID: uuid.New(), UserID: userID, Status: model.ApplicationApproved, DefaultMinuteLimit: 10, DefaultDailyLimit: 100}
	store := &fakeTokenStore{activeTokenCount: 2}
	svc := NewService(cfg, store, fakeTokenAppStore{app: app}, nil)

	_, err := svc.Create(context.Background(), userID, false, "prod")
	if err != model.ErrTokenLimitExceeded {
		t.Fatalf("expected token limit error, got %v", err)
	}
	if store.createMaxActiveTokens != 2 {
		t.Fatalf("expected max token limit 2, got %d", store.createMaxActiveTokens)
	}
	if store.created != nil {
		t.Fatal("token over the active limit must not be created")
	}
}

func TestAdminCanCreateTokenWithoutSubmittedApplication(t *testing.T) {
	cfg := config.Config{APITokenPrefix: "tgal_live", APITokenPepper: "pepper", DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}
	userID := uuid.New()
	adminApp := &model.Application{ID: uuid.New(), Status: model.ApplicationApproved}
	store := &fakeTokenStore{}
	svc := NewService(cfg, store, fakeTokenAppStore{ensuredAdmin: adminApp}, nil)

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

func TestListAdminNormalizesPageAndLimit(t *testing.T) {
	store := &fakeTokenStore{}
	svc := NewService(config.Config{}, store, fakeTokenAppStore{}, nil)
	if _, err := svc.ListAdmin(context.Background(), model.TokenActive, 0, 150); err != nil {
		t.Fatalf("ListAdmin returned error: %v", err)
	}
	if store.listStatus != model.TokenActive {
		t.Fatalf("unexpected status filter: %q", store.listStatus)
	}
	if store.listPage != 1 || store.listLimit != maxAdminListLimit {
		t.Fatalf("unexpected pagination: page=%d limit=%d", store.listPage, store.listLimit)
	}
}

func TestListAdminRejectsPageAboveCap(t *testing.T) {
	store := &fakeTokenStore{}
	svc := NewService(config.Config{}, store, fakeTokenAppStore{}, nil)
	if _, err := svc.ListAdmin(context.Background(), "", maxAdminListPage+1, 20); err != model.ErrInvalidInput {
		t.Fatalf("expected page cap error, got %v", err)
	}
	if store.listPage != 0 || store.listLimit != 0 {
		t.Fatal("invalid admin pagination must not reach the store")
	}
}

func TestUpdateNameMineTrimsAndScopesToken(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{}
	svc := NewService(config.Config{}, store, fakeTokenAppStore{}, nil)

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
	svc := NewService(config.Config{}, &fakeTokenStore{}, fakeTokenAppStore{}, nil)
	if _, err := svc.UpdateNameMine(context.Background(), uuid.New(), uuid.New(), "   "); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid input for blank token name, got %v", err)
	}
}

func TestDeleteMineRemovesTokenRecord(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{}
	svc := NewService(config.Config{}, store, fakeTokenAppStore{}, nil)

	if err := svc.DeleteMine(context.Background(), tokenID, userID); err != nil {
		t.Fatalf("delete token: %v", err)
	}
	if store.deletedForUserID != tokenID || store.deletedUserID != userID {
		t.Fatal("delete must remove the token row scoped to the current user")
	}
}

func validRawToken() string {
	return "tgal_live_" + strings.Repeat("A", generatedTokenSecretLength)
}

func authConfig() config.Config {
	return config.Config{APITokenPrefix: "tgal_live", APITokenPepper: "pepper", APITokenAuthCacheTTLSeconds: 60}
}

func TestAuthenticateRejectsDeletedToken(t *testing.T) {
	cfg := authConfig()
	raw := validRawToken()
	store := &fakeTokenStore{}
	svc := NewService(cfg, store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected deleted token unauthorized, got %v", err)
	}
}

func TestAuthenticateDeletesInactiveTokenRecord(t *testing.T) {
	cfg := authConfig()
	raw := validRawToken()
	tokenID := uuid.New()
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{Token: model.APIToken{ID: tokenID, Status: "disabled", TokenHash: HashAPIToken(raw, "pepper")}, ApplicationStatus: model.ApplicationApproved}}
	svc := NewService(cfg, store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected inactive token unauthorized, got %v", err)
	}
	if store.deletedID != tokenID {
		t.Fatal("inactive token record must be deleted instead of kept as disabled/revoked")
	}
}

func TestAuthenticateRejectsTokenAfterUserDisabledAndCacheInvalidated(t *testing.T) {
	raw := validRawToken()
	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{
		Token: model.APIToken{
			ID:        tokenID,
			UserID:    userID,
			Status:    model.TokenActive,
			TokenHash: HashAPIToken(raw, "pepper"),
		},
		ApplicationStatus: model.ApplicationApproved,
		UserStatus:        model.UserStatusActive,
	}}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("authenticate active user: %v", err)
	}

	store.auth.UserStatus = model.UserStatusDisabled
	if err := svc.InvalidateUser(context.Background(), userID); err != nil {
		t.Fatalf("invalidate user: %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected disabled user token unauthorized after cache invalidation, got %v", err)
	}
	if store.deletedID != uuid.Nil {
		t.Fatal("disabled user auth must not delete the token record")
	}
	if store.authLookups != 2 {
		t.Fatalf("expected cache miss after user invalidation, got %d lookups", store.authLookups)
	}
}

func TestAuthenticateDeletesExpiredTokenRecord(t *testing.T) {
	cfg := authConfig()
	raw := validRawToken()
	tokenID := uuid.New()
	expiredAt := time.Now().Add(-time.Second)
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{Token: model.APIToken{ID: tokenID, Status: model.TokenActive, TokenHash: HashAPIToken(raw, "pepper"), ExpiresAt: &expiredAt}, ApplicationStatus: model.ApplicationApproved, UserStatus: model.UserStatusActive}}
	svc := NewService(cfg, store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected expired token unauthorized, got %v", err)
	}
	if store.deletedID != tokenID {
		t.Fatal("expired token record must be deleted instead of kept in the database")
	}
}

func TestAuthenticateRejectsMalformedTokenWithoutStoreLookup(t *testing.T) {
	store := &fakeTokenStore{}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), "not-a-generated-token"); err != model.ErrUnauthorized {
		t.Fatalf("expected malformed token unauthorized, got %v", err)
	}
	if store.authLookups != 0 {
		t.Fatal("malformed token must not hit the token store")
	}
}

func TestAuthenticateCachesValidTokenAndDoesNotUpdateLastUsed(t *testing.T) {
	raw := validRawToken()
	tokenID := uuid.New()
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{
		Token:             model.APIToken{ID: tokenID, Status: model.TokenActive, TokenHash: HashAPIToken(raw, "pepper"), MinuteLimit: 10, DailyLimit: 100},
		ApplicationStatus: model.ApplicationApproved,
		UserStatus:        model.UserStatusActive,
	}}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("first authenticate: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("second authenticate: %v", err)
	}
	if store.authLookups != 1 {
		t.Fatalf("expected one store lookup, got %d", store.authLookups)
	}
	if store.lastUsedUpdates != 0 {
		t.Fatal("authenticate must not update last_used_at")
	}
}
func TestAuthenticateBoundsAuthCacheEntries(t *testing.T) {
	store := &fakeTokenStore{}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, nil)
	svc.authCacheMaxEntries = 2

	for i, ch := range []string{"A", "B", "C"} {
		raw := "tgal_live_" + strings.Repeat(ch, generatedTokenSecretLength)
		store.auth = &model.TokenAuthInfo{
			Token: model.APIToken{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				ApplicationID: uuid.New(),
				Status:        model.TokenActive,
				TokenHash:     HashAPIToken(raw, "pepper"),
				MinuteLimit:   10,
				DailyLimit:    100,
			},
			ApplicationStatus: model.ApplicationApproved,
			UserStatus:        model.UserStatusActive,
		}
		if _, err := svc.Authenticate(context.Background(), raw); err != nil {
			t.Fatalf("authenticate %d: %v", i, err)
		}
	}

	if len(svc.authCache) != 2 {
		t.Fatalf("expected auth cache to stay capped at 2 entries, got %d", len(svc.authCache))
	}
	for tokenID, tokenHash := range svc.authCacheTokenHashes {
		if _, ok := svc.authCache[tokenHash]; !ok {
			t.Fatalf("token index %s points at missing cache hash %s", tokenID, tokenHash)
		}
	}
}

func TestDeleteMineInvalidatesCachedToken(t *testing.T) {
	raw := validRawToken()
	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{
		Token:             model.APIToken{ID: tokenID, UserID: userID, ApplicationID: uuid.New(), Status: model.TokenActive, TokenHash: HashAPIToken(raw, "pepper"), MinuteLimit: 10, DailyLimit: 100},
		ApplicationStatus: model.ApplicationApproved,
		UserStatus:        model.UserStatusActive,
	}}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if err := svc.DeleteMine(context.Background(), tokenID, userID); err != nil {
		t.Fatalf("delete mine: %v", err)
	}
	store.auth = nil
	if _, err := svc.Authenticate(context.Background(), raw); err != model.ErrUnauthorized {
		t.Fatalf("expected deleted token cache invalidation, got %v", err)
	}
	if store.authLookups != 2 {
		t.Fatalf("expected cache miss after delete, got %d lookups", store.authLookups)
	}
}

func TestInvalidateUserAndApplicationEvictCachedTokens(t *testing.T) {
	raw := validRawToken()
	tokenID := uuid.New()
	userID := uuid.New()
	applicationID := uuid.New()
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{
		Token:             model.APIToken{ID: tokenID, UserID: userID, ApplicationID: applicationID, Status: model.TokenActive, TokenHash: HashAPIToken(raw, "pepper"), MinuteLimit: 10, DailyLimit: 100},
		ApplicationStatus: model.ApplicationApproved,
		UserStatus:        model.UserStatusActive,
	}}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, nil)
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if err := svc.InvalidateUser(context.Background(), userID); err != nil {
		t.Fatalf("invalidate user: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("authenticate after user invalidation: %v", err)
	}
	if err := svc.InvalidateApplication(context.Background(), applicationID); err != nil {
		t.Fatalf("invalidate application: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), raw); err != nil {
		t.Fatalf("authenticate after application invalidation: %v", err)
	}
	if store.authLookups != 3 {
		t.Fatalf("expected one lookup per invalidation, got %d", store.authLookups)
	}
}

func TestSharedAuthCacheEpochInvalidatesOtherServiceInstances(t *testing.T) {
	server := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer clientA.Close()
	clientB := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer clientB.Close()

	ctx := context.Background()
	raw := validRawToken()
	tokenID := uuid.New()
	userID := uuid.New()
	applicationID := uuid.New()
	tokenHash := HashAPIToken(raw, "pepper")
	authInfo := func(userStatus, applicationStatus string) *model.TokenAuthInfo {
		return &model.TokenAuthInfo{
			Token: model.APIToken{
				ID:            tokenID,
				UserID:        userID,
				ApplicationID: applicationID,
				Status:        model.TokenActive,
				TokenHash:     tokenHash,
				MinuteLimit:   10,
				DailyLimit:    100,
			},
			UserID:                 userID,
			UserStatus:             userStatus,
			UserMinuteLimit:        10,
			UserDailyLimit:         100,
			ApplicationID:          applicationID,
			ApplicationStatus:      applicationStatus,
			ApplicationMinuteLimit: 10,
			ApplicationDailyLimit:  100,
		}
	}

	storeA := &fakeTokenStore{}
	storeB := &fakeTokenStore{auth: authInfo(model.UserStatusActive, model.ApplicationApproved)}
	svcA := NewService(authConfig(), storeA, fakeTokenAppStore{}, clientA)
	svcB := NewService(authConfig(), storeB, fakeTokenAppStore{}, clientB)

	if _, err := svcB.Authenticate(ctx, raw); err != nil {
		t.Fatalf("initial authenticate: %v", err)
	}
	storeB.auth = authInfo(model.UserStatusDisabled, model.ApplicationApproved)
	if err := svcA.InvalidateUser(ctx, userID); err != nil {
		t.Fatalf("cross-instance invalidate user: %v", err)
	}
	if _, err := svcB.Authenticate(ctx, raw); err != model.ErrUnauthorized {
		t.Fatalf("expected disabled user to miss cross-instance cache, got %v", err)
	}

	storeB.auth = authInfo(model.UserStatusActive, model.ApplicationApproved)
	if _, err := svcB.Authenticate(ctx, raw); err != nil {
		t.Fatalf("authenticate after user re-enable: %v", err)
	}
	storeB.auth = authInfo(model.UserStatusActive, model.ApplicationRevoked)
	if err := svcA.InvalidateApplication(ctx, applicationID); err != nil {
		t.Fatalf("cross-instance invalidate application: %v", err)
	}
	if _, err := svcB.Authenticate(ctx, raw); err != model.ErrUnauthorized {
		t.Fatalf("expected revoked application to miss cross-instance cache, got %v", err)
	}

	storeB.auth = authInfo(model.UserStatusActive, model.ApplicationApproved)
	if _, err := svcB.Authenticate(ctx, raw); err != nil {
		t.Fatalf("authenticate before delete: %v", err)
	}
	if err := svcA.DeleteMine(ctx, tokenID, userID); err != nil {
		t.Fatalf("delete mine: %v", err)
	}
	storeB.auth = nil
	if _, err := svcB.Authenticate(ctx, raw); err != model.ErrUnauthorized {
		t.Fatalf("expected deleted token to miss cross-instance cache, got %v", err)
	}
	if storeB.authLookups != 6 {
		t.Fatalf("expected every shared invalidation to force a store lookup, got %d", storeB.authLookups)
	}
}

func TestMissingSharedAuthCacheEpochInvalidatesExistingEntries(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	ctx := context.Background()
	raw := validRawToken()
	tokenID := uuid.New()
	userID := uuid.New()
	tokenHash := HashAPIToken(raw, "pepper")
	store := &fakeTokenStore{auth: &model.TokenAuthInfo{
		Token:             model.APIToken{ID: tokenID, UserID: userID, ApplicationID: uuid.New(), Status: model.TokenActive, TokenHash: tokenHash, MinuteLimit: 10, DailyLimit: 100},
		ApplicationStatus: model.ApplicationApproved,
		UserStatus:        model.UserStatusActive,
	}}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, client)

	if _, err := svc.Authenticate(ctx, raw); err != nil {
		t.Fatalf("initial authenticate: %v", err)
	}
	oldEpoch, err := strconv.ParseInt(client.Get(ctx, authCacheEpochKey).Val(), 10, 64)
	if err != nil {
		t.Fatalf("parse initial auth cache epoch: %v", err)
	}
	server.Del(authCacheEpochKey)
	for time.Now().UnixNano() <= oldEpoch {
	}
	store.auth.UserStatus = model.UserStatusDisabled
	if _, err := svc.Authenticate(ctx, raw); err != model.ErrUnauthorized {
		t.Fatalf("expected missing shared epoch to force cache miss, got %v", err)
	}
	if store.authLookups != 2 {
		t.Fatalf("expected DB lookup after missing shared epoch, got %d", store.authLookups)
	}
}

func TestDeleteMineReturnsSharedInvalidationError(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	tokenID := uuid.New()
	userID := uuid.New()
	store := &fakeTokenStore{}
	svc := NewService(authConfig(), store, fakeTokenAppStore{}, client)
	server.Close()

	if err := svc.DeleteMine(context.Background(), tokenID, userID); err == nil {
		t.Fatal("expected delete to surface shared auth cache invalidation failure")
	}
	if store.deletedForUserID != tokenID || store.deletedUserID != userID {
		t.Fatal("delete must still remove the token row before reporting invalidation failure")
	}
}
