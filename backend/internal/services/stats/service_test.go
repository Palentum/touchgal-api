package stats

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeDashboardStore struct {
	calls    int
	lastDays int
	data     model.StatsDashboard
}

func (s *fakeDashboardStore) Dashboard(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsDashboard, error) {
	s.calls++
	s.lastDays = days
	return s.data, nil
}

func TestDashboardCachesSameNormalizedKey(t *testing.T) {
	store := &fakeDashboardStore{data: model.StatsDashboard{Summary: model.StatsSummary{TotalRequests: 7}}}
	svc := NewService(store)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc.nowFunc = func() time.Time { return now }
	userID := uuid.New()

	first, err := svc.Dashboard(context.Background(), userID, 999, nil)
	if err != nil {
		t.Fatalf("first dashboard: %v", err)
	}
	second, err := svc.Dashboard(context.Background(), userID, 90, nil)
	if err != nil {
		t.Fatalf("second dashboard: %v", err)
	}
	if store.calls != 1 {
		t.Fatalf("expected cached dashboard call, got %d store calls", store.calls)
	}
	if store.lastDays != 90 {
		t.Fatalf("expected normalized days 90, got %d", store.lastDays)
	}
	if first.Summary.TotalRequests != 7 || second.Summary.TotalRequests != 7 {
		t.Fatalf("unexpected dashboard data: %#v %#v", first, second)
	}
}

func TestDashboardCacheSeparatesToken(t *testing.T) {
	store := &fakeDashboardStore{}
	svc := NewService(store)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc.nowFunc = func() time.Time { return now }
	userID := uuid.New()
	tokenID := uuid.New()

	if _, err := svc.Dashboard(context.Background(), userID, 30, nil); err != nil {
		t.Fatalf("all tokens dashboard: %v", err)
	}
	if _, err := svc.Dashboard(context.Background(), userID, 30, &tokenID); err != nil {
		t.Fatalf("token dashboard: %v", err)
	}
	if _, err := svc.Dashboard(context.Background(), userID, 30, &tokenID); err != nil {
		t.Fatalf("cached token dashboard: %v", err)
	}
	if store.calls != 2 {
		t.Fatalf("expected nil token and token keys to be separate, got %d store calls", store.calls)
	}
}

func TestDashboardCacheExpires(t *testing.T) {
	store := &fakeDashboardStore{data: model.StatsDashboard{Summary: model.StatsSummary{TotalRequests: 1}}}
	svc := NewService(store)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc.nowFunc = func() time.Time { return now }
	userID := uuid.New()

	first, err := svc.Dashboard(context.Background(), userID, 30, nil)
	if err != nil {
		t.Fatalf("first dashboard: %v", err)
	}
	store.data = model.StatsDashboard{Summary: model.StatsSummary{TotalRequests: 2}}
	now = now.Add(dashboardCacheTTL + time.Nanosecond)
	second, err := svc.Dashboard(context.Background(), userID, 30, nil)
	if err != nil {
		t.Fatalf("second dashboard: %v", err)
	}
	if store.calls != 2 {
		t.Fatalf("expected cache refresh after ttl, got %d store calls", store.calls)
	}
	if first.Summary.TotalRequests != 1 || second.Summary.TotalRequests != 2 {
		t.Fatalf("unexpected dashboard data across ttl: %#v %#v", first, second)
	}
}

func TestDashboardCachePrunesExpiredEntries(t *testing.T) {
	store := &fakeDashboardStore{}
	svc := NewService(store)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc.nowFunc = func() time.Time { return now }

	if _, err := svc.Dashboard(context.Background(), uuid.New(), 30, nil); err != nil {
		t.Fatalf("first dashboard: %v", err)
	}
	now = now.Add(dashboardCacheTTL + time.Nanosecond)
	if _, err := svc.Dashboard(context.Background(), uuid.New(), 30, nil); err != nil {
		t.Fatalf("second dashboard: %v", err)
	}
	if len(svc.cache) != 1 {
		t.Fatalf("expected expired cache entries to be pruned, got %d entries", len(svc.cache))
	}
}

func TestDashboardCacheDoesNotGrowPastMaxEntries(t *testing.T) {
	store := &fakeDashboardStore{}
	svc := NewService(store)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc.nowFunc = func() time.Time { return now }

	var lastUserID uuid.UUID
	for i := 0; i < dashboardCacheMaxEntries+1; i++ {
		lastUserID = uuid.New()
		if _, err := svc.Dashboard(context.Background(), lastUserID, 30, nil); err != nil {
			t.Fatalf("dashboard %d: %v", i, err)
		}
	}
	if len(svc.cache) != dashboardCacheMaxEntries {
		t.Fatalf("expected cache cap %d, got %d", dashboardCacheMaxEntries, len(svc.cache))
	}
	callsAfterFill := store.calls
	if _, err := svc.Dashboard(context.Background(), lastUserID, 30, nil); err != nil {
		t.Fatalf("cached latest dashboard: %v", err)
	}
	if store.calls != callsAfterFill {
		t.Fatal("cache must keep accepting new entries after reaching capacity")
	}
}

func TestDashboardCacheEntryRefreshDoesNotEvictExistingKey(t *testing.T) {
	target := newDashboardCacheKey(uuid.New(), 30, nil)
	sentinel := newDashboardCacheKey(uuid.New(), 30, nil)
	cache := make(map[dashboardCacheKey]dashboardCacheEntry, dashboardCacheMaxEntries)
	entry := dashboardCacheEntry{expiresAt: time.Now().Add(time.Minute)}
	cache[target] = entry
	cache[sentinel] = entry
	for len(cache) < dashboardCacheMaxEntries {
		cache[newDashboardCacheKey(uuid.New(), 30, nil)] = entry
	}

	cacheDashboardEntryLocked(cache, target, dashboardCacheEntry{
		data:      model.StatsDashboard{Summary: model.StatsSummary{TotalRequests: 99}},
		expiresAt: entry.expiresAt,
	})

	if len(cache) != dashboardCacheMaxEntries {
		t.Fatalf("expected cache size to remain capped, got %d", len(cache))
	}
	if _, ok := cache[sentinel]; !ok {
		t.Fatal("refreshing an existing key must not evict another entry")
	}
	if cache[target].data.Summary.TotalRequests != 99 {
		t.Fatalf("expected target entry to be refreshed, got %#v", cache[target].data)
	}
}
