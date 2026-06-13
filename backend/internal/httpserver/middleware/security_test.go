package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersApplyToJSONRoutes(t *testing.T) {
	handler := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assertSecurityHeaders(t, res, apiContentSecurityPolicy)
}

func TestSecurityHeadersAllowPinnedDocsScript(t *testing.T) {
	handler := SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assertSecurityHeaders(t, res, docsContentSecurityPolicy)
	if csp := res.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "https://cdn.jsdelivr.net") {
		t.Fatalf("expected docs CSP to allow Scalar CDN, got %q", csp)
	}
}

func TestNoStoreSetsSensitiveCacheHeaders(t *testing.T) {
	handler := NoStore(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/tokens", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", got)
	}
	if got := res.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("expected Pragma no-cache, got %q", got)
	}
	if got := res.Header().Get("Expires"); got != "0" {
		t.Fatalf("expected Expires 0, got %q", got)
	}
}

func assertSecurityHeaders(t *testing.T, res *httptest.ResponseRecorder, wantCSP string) {
	t.Helper()
	checks := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": wantCSP,
	}
	for header, want := range checks {
		if got := res.Header().Get(header); got != want {
			t.Fatalf("expected %s %q, got %q", header, want, got)
		}
	}
	if got := res.Header().Get("Permissions-Policy"); got == "" {
		t.Fatal("expected Permissions-Policy header")
	}
}
