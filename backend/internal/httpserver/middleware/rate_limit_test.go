package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestAPIRateLimit(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	info := &model.TokenAuthInfo{Token: model.APIToken{ID: uuid.New(), MinuteLimit: 1, DailyLimit: 10}}
	handler := APIRateLimit(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
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
