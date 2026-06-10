package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestObservabilityRouterMountsEnabledHandlersOnly(t *testing.T) {
	router := NewObservabilityRouter(true, false)
	pprofRes := httptest.NewRecorder()
	router.ServeHTTP(pprofRes, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if pprofRes.Code != http.StatusOK {
		t.Fatalf("expected pprof index to be mounted, got %d", pprofRes.Code)
	}
	metricsRes := httptest.NewRecorder()
	router.ServeHTTP(metricsRes, httptest.NewRequest(http.MethodGet, "/debug/vars", nil))
	if metricsRes.Code != http.StatusNotFound {
		t.Fatalf("expected metrics to be disabled, got %d", metricsRes.Code)
	}

	router = NewObservabilityRouter(false, true)
	metricsRes = httptest.NewRecorder()
	router.ServeHTTP(metricsRes, httptest.NewRequest(http.MethodGet, "/debug/vars", nil))
	if metricsRes.Code != http.StatusOK {
		t.Fatalf("expected metrics to be mounted, got %d", metricsRes.Code)
	}
}
