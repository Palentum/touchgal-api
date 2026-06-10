package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeLastUsedService struct {
	calls chan uuid.UUID
}

func (f fakeLastUsedService) MarkLastUsed(ctx context.Context, id uuid.UUID) error {
	f.calls <- id
	return nil
}

func TestAPIPreAuthRateLimitBlocksBeforeNext(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	called := 0
	handler := APIPreAuthRateLimit(client, 1, 10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))
	request := func() int {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.RemoteAddr = "203.0.113.10:12345"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res.Code
	}
	if code := request(); code != http.StatusNoContent {
		t.Fatalf("first request got %d", code)
	}
	if code := request(); code != http.StatusTooManyRequests {
		t.Fatalf("second request got %d", code)
	}
	if called != 1 {
		t.Fatalf("next handler called %d times", called)
	}
}

func TestAPIRateLimit(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	info := &model.TokenAuthInfo{Token: model.APIToken{ID: uuid.New(), MinuteLimit: 1, DailyLimit: 10}}
	called := 0
	handler := APIRateLimit(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))
	request := func() int {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req = req.WithContext(WithTokenInfo(req.Context(), info))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res.Code
	}
	if code := request(); code != http.StatusNoContent {
		t.Fatalf("first request got %d", code)
	}
	if code := request(); code != http.StatusTooManyRequests {
		t.Fatalf("second request got %d", code)
	}
	if called != 1 {
		t.Fatalf("next handler called %d times", called)
	}
}

func TestAPILastUsedThrottlesDatabaseUpdates(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	tokenID := uuid.New()
	service := fakeLastUsedService{calls: make(chan uuid.UUID, 2)}
	handler := APILastUsed(service, client, zerolog.Nop(), time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := func() int {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req = req.WithContext(WithTokenInfo(req.Context(), &model.TokenAuthInfo{Token: model.APIToken{ID: tokenID}}))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res.Code
	}
	if code := request(); code != http.StatusNoContent {
		t.Fatalf("first request got %d", code)
	}
	select {
	case got := <-service.calls:
		if got != tokenID {
			t.Fatalf("last used token ID got %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("last used update was not called")
	}
	if code := request(); code != http.StatusNoContent {
		t.Fatalf("second request got %d", code)
	}
	select {
	case <-service.calls:
		t.Fatal("last used update must be throttled")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestAPIRateLimitAppliesUserCap(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	info := &model.TokenAuthInfo{
		Token:           model.APIToken{ID: uuid.New(), MinuteLimit: 10, DailyLimit: 100},
		UserMinuteLimit: 1,
		UserDailyLimit:  10,
	}
	handler := APIRateLimit(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req = req.WithContext(WithTokenInfo(req.Context(), info))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	first := request()
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request got %d", first.Code)
	}
	if got := first.Header().Get("X-RateLimit-Limit-Minute"); got != "1" {
		t.Fatalf("minute limit header got %q", got)
	}
	if code := request().Code; code != http.StatusTooManyRequests {
		t.Fatalf("second request got %d", code)
	}
}
