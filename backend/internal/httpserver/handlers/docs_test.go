package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsOpenAPIServesEmbeddedDocumentWithCacheHeaders(t *testing.T) {
	handler := DocsHandler{}
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	handler.OpenAPI(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/yaml; charset=utf-8" {
		t.Fatalf("expected yaml content type, got %q", got)
	}
	assertDocsCacheHeaders(t, rec)
	if body := rec.Body.String(); !strings.Contains(body, "openapi: 3.1.0") || !strings.Contains(body, "title: TouchGal API") {
		t.Fatalf("expected embedded OpenAPI document, got %q", body)
	}
}

func TestDocsOpenAPIReturnsNotModifiedForMatchingETag(t *testing.T) {
	handler := DocsHandler{}
	first := httptest.NewRecorder()
	handler.OpenAPI(first, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	req.Header.Set("If-None-Match", "W/"+etag)
	rec := httptest.NewRecorder()

	handler.OpenAPI(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, rec.Code)
	}
	assertDocsCacheHeaders(t, rec)
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty 304 body, got %q", rec.Body.String())
	}
}

func TestDocsSwaggerServesPinnedScalarPageWithCacheHeaders(t *testing.T) {
	handler := DocsHandler{}
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	handler.Swagger(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected html content type, got %q", got)
	}
	assertDocsCacheHeaders(t, rec)
	body := rec.Body.String()
	if !strings.Contains(body, "@scalar/api-reference@1.59.2") {
		t.Fatalf("expected pinned Scalar script, got %q", body)
	}
	if !strings.Contains(body, `configuration='{"url":"/openapi.yaml"}'`) {
		t.Fatalf("expected OpenAPI configuration, got %q", body)
	}
}

func assertDocsCacheHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if got := rec.Header().Get("Cache-Control"); got != docsCacheControl {
		t.Fatalf("expected Cache-Control %q, got %q", docsCacheControl, got)
	}
	etag := rec.Header().Get("ETag")
	if !strings.HasPrefix(etag, `"sha256-`) || !strings.HasSuffix(etag, `"`) {
		t.Fatalf("expected sha256 ETag, got %q", etag)
	}
}
