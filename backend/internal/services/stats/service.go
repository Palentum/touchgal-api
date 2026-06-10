package stats

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

const (
	dashboardCacheTTL        = 30 * time.Second
	dashboardCacheMaxEntries = 4096
)

type Store interface {
	Dashboard(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsDashboard, error)
}

type Service struct {
	store    Store
	cacheMu  sync.RWMutex
	cache    map[dashboardCacheKey]dashboardCacheEntry
	cacheTTL time.Duration
	nowFunc  func() time.Time
}

type dashboardCacheKey struct {
	userID   uuid.UUID
	days     int
	tokenID  uuid.UUID
	hasToken bool
}

type dashboardCacheEntry struct {
	data      model.StatsDashboard
	expiresAt time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store:    store,
		cache:    make(map[dashboardCacheKey]dashboardCacheEntry),
		cacheTTL: dashboardCacheTTL,
		nowFunc:  time.Now,
	}
}

func NormalizeDays(days int) int {
	if days <= 0 {
		return 30
	}
	if days > 90 {
		return 90
	}
	return days
}

func (s *Service) Dashboard(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsDashboard, error) {
	normalizedDays := NormalizeDays(days)
	key := newDashboardCacheKey(userID, normalizedDays, tokenID)
	now := s.nowFunc()

	s.cacheMu.RLock()
	entry, ok := s.cache[key]
	if ok && now.Before(entry.expiresAt) {
		s.cacheMu.RUnlock()
		return entry.data, nil
	}
	s.cacheMu.RUnlock()

	data, err := s.store.Dashboard(ctx, userID, normalizedDays, tokenID)
	if err != nil {
		return model.StatsDashboard{}, err
	}

	if s.cacheTTL > 0 {
		cacheNow := s.nowFunc()
		expiresAt := cacheNow.Add(s.cacheTTL)
		s.cacheMu.Lock()
		if s.cache == nil {
			s.cache = make(map[dashboardCacheKey]dashboardCacheEntry)
		}
		pruneExpiredDashboardCacheLocked(s.cache, cacheNow)
		if len(s.cache) < dashboardCacheMaxEntries {
			s.cache[key] = dashboardCacheEntry{data: data, expiresAt: expiresAt}
		}
		s.cacheMu.Unlock()
	}
	return data, nil
}

func newDashboardCacheKey(userID uuid.UUID, days int, tokenID *uuid.UUID) dashboardCacheKey {
	key := dashboardCacheKey{userID: userID, days: days}
	if tokenID != nil {
		key.tokenID = *tokenID
		key.hasToken = true
	}
	return key
}

func pruneExpiredDashboardCacheLocked(cache map[dashboardCacheKey]dashboardCacheEntry, now time.Time) {
	for key, entry := range cache {
		if !now.Before(entry.expiresAt) {
			delete(cache, key)
		}
	}
}
