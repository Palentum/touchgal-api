package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/touchgal/developer/backend/internal/model"
)

type fakeSyncStarter struct {
	called bool
}

func (f *fakeSyncStarter) Start(ctx context.Context, mode string) (*model.SyncRun, error) {
	f.called = true
	return &model.SyncRun{Mode: mode}, nil
}

func TestAdminRunSyncRejectsWhenSyncDisabled(t *testing.T) {
	starter := &fakeSyncStarter{}
	handler := NewAdminHandler(nil, nil, nil, starter, nil, false)

	req := httptest.NewRequest(http.MethodPost, "/admin/sync/runs", strings.NewReader(`{"mode":"incremental"}`))
	rec := httptest.NewRecorder()

	handler.RunSync(rec, req)

	if starter.called {
		t.Fatal("expected sync service not to be called")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"SYNC_DISABLED"`) {
		t.Fatalf("expected SYNC_DISABLED response, got %s", rec.Body.String())
	}
}

func TestAdminRunSyncRejectsWhenSyncServiceMissing(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil, nil, nil, true)

	req := httptest.NewRequest(http.MethodPost, "/admin/sync/runs", strings.NewReader(`{"mode":"incremental"}`))
	rec := httptest.NewRecorder()

	handler.RunSync(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"SYNC_DISABLED"`) {
		t.Fatalf("expected SYNC_DISABLED response, got %s", rec.Body.String())
	}
}

func TestAdminRunSyncRejectsLargeBody(t *testing.T) {
	starter := &fakeSyncStarter{}
	handler := NewAdminHandler(nil, nil, nil, starter, nil, true)
	body := `{"mode":"` + strings.Repeat("x", smallJSONBodyLimit) + `"}`

	req := httptest.NewRequest(http.MethodPost, "/admin/sync/runs", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.RunSync(rec, req)

	if starter.called {
		t.Fatal("expected sync service not to be called")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"REQUEST_BODY_TOO_LARGE"`) {
		t.Fatalf("expected REQUEST_BODY_TOO_LARGE response, got %s", rec.Body.String())
	}
}
