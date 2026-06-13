package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type Store interface {
	Create(ctx context.Context, token model.APIToken, maxActiveTokensPerUser int) (*model.APIToken, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error)
	ListAdmin(ctx context.Context, status string, page, limit int) ([]model.APIToken, error)
	GetByHashWithApplication(ctx context.Context, tokenHash string) (*model.TokenAuthInfo, error)
	DeleteForUser(ctx context.Context, id, userID uuid.UUID) error
	UpdateNameForUser(ctx context.Context, id, userID uuid.UUID, name string) (*model.APIToken, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	CountActive(ctx context.Context) (int, error)
}

type ApplicationStore interface {
	GetApprovedByUser(ctx context.Context, userID uuid.UUID) (*model.Application, error)
	EnsureAdminApproved(ctx context.Context, userID uuid.UUID, minuteLimit, dailyLimit int) (*model.Application, error)
}

type Service struct {
	cfg          config.Config
	tokens       Store
	applications ApplicationStore
	redis        *redis.Client

	authCacheTTL               time.Duration
	authCacheMaxEntries        int
	authCacheMu                sync.RWMutex
	authCache                  map[string]cachedAuthInfo
	authCacheTokenHashes       map[uuid.UUID]string
	authCacheUserTokens        map[uuid.UUID]map[uuid.UUID]struct{}
	authCacheApplicationTokens map[uuid.UUID]map[uuid.UUID]struct{}
}

type cachedAuthInfo struct {
	info           model.TokenAuthInfo
	expiresAt      time.Time
	authCacheEpoch string
}

type CreateResult struct {
	Token      *model.APIToken `json:"tokenRecord"`
	PlainToken string          `json:"token"`
}

func NewService(cfg config.Config, tokens Store, applications ApplicationStore, redisClient *redis.Client) *Service {
	authCacheTTL := cfg.APITokenAuthCacheTTL()
	if authCacheTTL <= 0 {
		authCacheTTL = time.Minute
	}
	authCacheMaxEntries := cfg.APITokenAuthCacheMaxEntries
	if authCacheMaxEntries <= 0 {
		authCacheMaxEntries = config.DefaultAPITokenAuthCacheMaxEntries
	}
	return &Service{
		cfg:                        cfg,
		tokens:                     tokens,
		applications:               applications,
		redis:                      redisClient,
		authCacheTTL:               authCacheTTL,
		authCacheMaxEntries:        authCacheMaxEntries,
		authCache:                  make(map[string]cachedAuthInfo),
		authCacheTokenHashes:       make(map[uuid.UUID]string),
		authCacheUserTokens:        make(map[uuid.UUID]map[uuid.UUID]struct{}),
		authCacheApplicationTokens: make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
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

const generatedTokenSecretLength = 43

func generatedTokenPrefix(prefix string) string {
	if prefix == "" {
		prefix = "tgal_live"
	}
	return prefix + "_"
}

func generatedTokenShape(raw, prefix string) bool {
	tokenPrefix := generatedTokenPrefix(prefix)
	if !strings.HasPrefix(raw, tokenPrefix) {
		return false
	}
	secret := raw[len(tokenPrefix):]
	if len(secret) != generatedTokenSecretLength {
		return false
	}
	for i := 0; i < len(secret); i++ {
		if !isRawURLEncodingChar(secret[i]) {
			return false
		}
	}
	return true
}

func isRawURLEncodingChar(c byte) bool {
	return c >= 'A' && c <= 'Z' ||
		c >= 'a' && c <= 'z' ||
		c >= '0' && c <= '9' ||
		c == '-' || c == '_'
}

func CanCreateFromApplication(app *model.Application) bool {
	return app != nil && app.Status == model.ApplicationApproved
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, isAdmin bool, name string) (*CreateResult, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return nil, model.ErrInvalidInput
	}
	app, err := s.applications.GetApprovedByUser(ctx, userID)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
		if !isAdmin {
			return nil, model.ErrApplicationOpen
		}
		app, err = s.applications.EnsureAdminApproved(ctx, userID, s.cfg.DefaultTokenMinuteLimit, s.cfg.DefaultTokenDailyLimit)
		if err != nil {
			return nil, err
		}
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
	created, err := s.tokens.Create(ctx, record, s.maxActiveTokensPerUser())
	if err != nil {
		return nil, err
	}
	return &CreateResult{Token: created, PlainToken: raw}, nil
}

func (s *Service) ListMine(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return s.tokens.ListByUser(ctx, userID)
}
func (s *Service) maxActiveTokensPerUser() int {
	if s.cfg.MaxActiveTokensPerUser > 0 {
		return s.cfg.MaxActiveTokensPerUser
	}
	return config.DefaultMaxActiveTokensPerUser
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
func (s *Service) UpdateNameMine(ctx context.Context, id, userID uuid.UUID, name string) (*model.APIToken, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return nil, model.ErrInvalidInput
	}
	return s.tokens.UpdateNameForUser(ctx, id, userID, name)
}

func (s *Service) DeleteMine(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.tokens.DeleteForUser(ctx, id, userID); err != nil {
		return err
	}
	return s.InvalidateToken(ctx, id)
}
func (s *Service) DeleteByAdmin(ctx context.Context, id uuid.UUID) error {
	if err := s.tokens.DeleteByID(ctx, id); err != nil {
		return err
	}
	return s.InvalidateToken(ctx, id)
}
func (s *Service) CountActive(ctx context.Context) (int, error) { return s.tokens.CountActive(ctx) }

func (s *Service) Authenticate(ctx context.Context, raw string) (*model.TokenAuthInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || !generatedTokenShape(raw, s.cfg.APITokenPrefix) {
		return nil, model.ErrUnauthorized
	}
	now := time.Now()
	tokenHash := HashAPIToken(raw, s.cfg.APITokenPepper)
	if info, ok := s.cachedAuthInfo(ctx, tokenHash, now); ok {
		return info, nil
	}
	if s.redis == nil {
		info, err := s.loadAuthInfo(ctx, tokenHash, now)
		if err != nil {
			return nil, err
		}
		s.cacheAuthInfo(tokenHash, info, now, "")
		return info, nil
	}
	for {
		epochBefore, beforeOK := s.currentAuthCacheEpoch(ctx)
		info, err := s.loadAuthInfo(ctx, tokenHash, now)
		if err != nil {
			return nil, err
		}
		epochAfter, afterOK := s.currentAuthCacheEpoch(ctx)
		if beforeOK && afterOK {
			if epochBefore != epochAfter {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				continue
			}
			s.cacheAuthInfo(tokenHash, info, now, epochAfter)
		}
		return info, nil
	}
}

func (s *Service) MarkLastUsed(ctx context.Context, id uuid.UUID) error {
	return s.tokens.UpdateLastUsed(ctx, id)
}

func (s *Service) loadAuthInfo(ctx context.Context, tokenHash string, now time.Time) (*model.TokenAuthInfo, error) {
	info, err := s.tokens.GetByHashWithApplication(ctx, tokenHash)
	if err != nil {
		return nil, model.ErrUnauthorized
	}
	if info.Token.Status != model.TokenActive {
		_ = s.tokens.DeleteByID(ctx, info.Token.ID)
		_ = s.InvalidateToken(ctx, info.Token.ID)
		return nil, model.ErrUnauthorized
	}
	if info.UserStatus != model.UserStatusActive {
		return nil, model.ErrUnauthorized
	}
	if info.ApplicationStatus != model.ApplicationApproved {
		return nil, model.ErrUnauthorized
	}
	if info.Token.ExpiresAt != nil && !now.Before(*info.Token.ExpiresAt) {
		_ = s.tokens.DeleteByID(ctx, info.Token.ID)
		_ = s.InvalidateToken(ctx, info.Token.ID)
		return nil, model.ErrUnauthorized
	}
	return info, nil
}

func (s *Service) InvalidateToken(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return nil
	}
	err := s.bumpAuthCacheEpoch(ctx)
	s.authCacheMu.Lock()
	s.evictTokenLocked(id)
	s.authCacheMu.Unlock()
	return err
}

func (s *Service) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return nil
	}
	err := s.bumpAuthCacheEpoch(ctx)
	s.authCacheMu.Lock()
	for tokenID := range s.authCacheUserTokens[userID] {
		s.evictTokenLocked(tokenID)
	}
	s.authCacheMu.Unlock()
	return err
}

func (s *Service) InvalidateApplication(ctx context.Context, applicationID uuid.UUID) error {
	if applicationID == uuid.Nil {
		return nil
	}
	err := s.bumpAuthCacheEpoch(ctx)
	s.authCacheMu.Lock()
	for tokenID := range s.authCacheApplicationTokens[applicationID] {
		s.evictTokenLocked(tokenID)
	}
	s.authCacheMu.Unlock()
	return err
}

const (
	authCacheEpochKey                 = "token:auth:epoch"
	authCacheInvalidationWriteTimeout = 3 * time.Second
)

func (s *Service) currentAuthCacheEpoch(ctx context.Context) (string, bool) {
	if s.redis == nil {
		return "", false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	epoch, err := s.redis.Get(ctx, authCacheEpochKey).Result()
	if errors.Is(err, redis.Nil) {
		return s.initializeAuthCacheEpoch(ctx)
	}
	if err != nil {
		return "", false
	}
	return epoch, true
}

func (s *Service) initializeAuthCacheEpoch(ctx context.Context) (string, bool) {
	epoch := strconv.FormatInt(time.Now().UnixNano(), 10)
	created, err := s.redis.SetNX(ctx, authCacheEpochKey, epoch, 0).Result()
	if err != nil {
		return "", false
	}
	if created {
		return epoch, true
	}
	epoch, err = s.redis.Get(ctx, authCacheEpochKey).Result()
	if err != nil {
		return "", false
	}
	return epoch, true
}

func (s *Service) bumpAuthCacheEpoch(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), authCacheInvalidationWriteTimeout)
	defer cancel()
	created, err := s.redis.SetNX(writeCtx, authCacheEpochKey, strconv.FormatInt(time.Now().UnixNano(), 10), 0).Result()
	if err != nil {
		return err
	}
	if created {
		return nil
	}
	return s.redis.Incr(writeCtx, authCacheEpochKey).Err()
}

func (s *Service) cachedAuthInfo(ctx context.Context, tokenHash string, now time.Time) (*model.TokenAuthInfo, bool) {
	s.authCacheMu.RLock()
	entry, ok := s.authCache[tokenHash]
	s.authCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if !now.Before(entry.expiresAt) {
		s.authCacheMu.Lock()
		if current, ok := s.authCache[tokenHash]; ok && !now.Before(current.expiresAt) {
			s.evictTokenLocked(current.info.Token.ID)
		}
		s.authCacheMu.Unlock()
		return nil, false
	}
	if s.redis != nil {
		currentEpoch, ok := s.currentAuthCacheEpoch(ctx)
		if !ok || currentEpoch != entry.authCacheEpoch {
			s.authCacheMu.Lock()
			if current, ok := s.authCache[tokenHash]; ok && current.info.Token.ID == entry.info.Token.ID {
				s.evictTokenLocked(current.info.Token.ID)
			}
			s.authCacheMu.Unlock()
			return nil, false
		}
	}
	info := entry.info
	return &info, true
}

func (s *Service) cacheAuthInfo(tokenHash string, info *model.TokenAuthInfo, now time.Time, authCacheEpoch string) {
	if s.authCacheTTL <= 0 || info == nil {
		return
	}
	expiresAt := now.Add(s.authCacheTTL)
	if info.Token.ExpiresAt != nil {
		if !now.Before(*info.Token.ExpiresAt) {
			return
		}
		if info.Token.ExpiresAt.Before(expiresAt) {
			expiresAt = *info.Token.ExpiresAt
		}
	}
	s.authCacheMu.Lock()
	if s.authCache == nil {
		s.authCache = make(map[string]cachedAuthInfo)
	}
	if s.authCacheTokenHashes == nil {
		s.authCacheTokenHashes = make(map[uuid.UUID]string)
	}
	if s.authCacheUserTokens == nil {
		s.authCacheUserTokens = make(map[uuid.UUID]map[uuid.UUID]struct{})
	}
	if s.authCacheApplicationTokens == nil {
		s.authCacheApplicationTokens = make(map[uuid.UUID]map[uuid.UUID]struct{})
	}
	s.evictTokenLocked(info.Token.ID)
	s.pruneExpiredAuthCacheLocked(now)
	if maxEntries := s.authCacheMaxEntries; maxEntries > 0 && len(s.authCache) >= maxEntries {
		s.evictAnyAuthCacheEntryLocked()
	}
	s.authCache[tokenHash] = cachedAuthInfo{info: *info, expiresAt: expiresAt, authCacheEpoch: authCacheEpoch}
	s.authCacheTokenHashes[info.Token.ID] = tokenHash
	addTokenIndex(s.authCacheUserTokens, info.Token.UserID, info.Token.ID)
	addTokenIndex(s.authCacheApplicationTokens, info.Token.ApplicationID, info.Token.ID)
	s.authCacheMu.Unlock()
}

func (s *Service) pruneExpiredAuthCacheLocked(now time.Time) {
	for tokenHash, entry := range s.authCache {
		if !now.Before(entry.expiresAt) {
			s.evictTokenLocked(entry.info.Token.ID)
			continue
		}
		if s.authCacheTokenHashes[entry.info.Token.ID] != tokenHash {
			delete(s.authCache, tokenHash)
		}
	}
}

func (s *Service) evictAnyAuthCacheEntryLocked() {
	for _, entry := range s.authCache {
		s.evictTokenLocked(entry.info.Token.ID)
		return
	}
}

func addTokenIndex(index map[uuid.UUID]map[uuid.UUID]struct{}, ownerID, tokenID uuid.UUID) {
	tokens := index[ownerID]
	if tokens == nil {
		tokens = make(map[uuid.UUID]struct{})
		index[ownerID] = tokens
	}
	tokens[tokenID] = struct{}{}
}

func (s *Service) evictTokenLocked(tokenID uuid.UUID) {
	tokenHash, ok := s.authCacheTokenHashes[tokenID]
	if !ok {
		return
	}
	entry, hasEntry := s.authCache[tokenHash]
	delete(s.authCache, tokenHash)
	delete(s.authCacheTokenHashes, tokenID)
	if !hasEntry {
		return
	}
	deleteTokenIndex(s.authCacheUserTokens, entry.info.Token.UserID, tokenID)
	deleteTokenIndex(s.authCacheApplicationTokens, entry.info.Token.ApplicationID, tokenID)
}

func deleteTokenIndex(index map[uuid.UUID]map[uuid.UUID]struct{}, ownerID, tokenID uuid.UUID) {
	tokens := index[ownerID]
	if tokens == nil {
		return
	}
	delete(tokens, tokenID)
	if len(tokens) == 0 {
		delete(index, ownerID)
	}
}
