package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadySucceedsWhenChecksPass(t *testing.T) {
	handler := newHealthHandlerWithChecks(readinessCheck{
		check: func(context.Context) error { return nil },
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	res := httptest.NewRecorder()

	handler.Ready(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"status":"ready"`) {
		t.Fatalf("expected ready response, got %s", res.Body.String())
	}
}

func TestReadyFailsWhenCheckFails(t *testing.T) {
	handler := newHealthHandlerWithChecks(readinessCheck{
		check: func(context.Context) error { return errors.New("redis unavailable") },
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	res := httptest.NewRecorder()

	handler.Ready(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}
	if !strings.Contains(res.Body.String(), `"code":"NOT_READY"`) {
		t.Fatalf("expected NOT_READY response, got %s", res.Body.String())
	}
}

func TestReadyFailsWithoutConfiguredChecks(t *testing.T) {
	handler := newHealthHandlerWithChecks()
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	res := httptest.NewRecorder()

	handler.Ready(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}
	if !strings.Contains(res.Body.String(), `"code":"NOT_READY"`) {
		t.Fatalf("expected NOT_READY response, got %s", res.Body.String())
	}
}
