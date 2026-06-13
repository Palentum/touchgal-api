package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/model"
	usersvc "github.com/touchgal/developer/backend/internal/services/users"
)

type fakeSyncStarter struct {
	called bool
}

func (f *fakeSyncStarter) Start(ctx context.Context, mode string) (*model.SyncRun, error) {
	f.called = true
	return &model.SyncRun{Mode: mode}, nil
}

type fakeAdminUserStore struct {
	updatedID uuid.UUID
}

func (f *fakeAdminUserStore) ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error) {
	return nil, nil
}

func (f *fakeAdminUserStore) UpdateAdmin(ctx context.Context, id uuid.UUID, email, displayName, status *string, minuteLimit, dailyLimit *int) (*model.User, error) {
	f.updatedID = id
	return &model.User{ID: id, Email: "updated@example.com", DisplayName: "updated", Status: model.UserStatusActive}, nil
}

func (f *fakeAdminUserStore) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestAdminUpdateUserProfileOnlySkipsTokenInvalidation(t *testing.T) {
	targetID := uuid.New()
	store := &fakeAdminUserStore{}
	handler := NewAdminHandler(nil, nil, usersvc.NewService(store), nil, nil, nil, false)
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/"+targetID.String(), strings.NewReader(`{"displayName":"renamed"}`))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", targetID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = middleware.WithUser(ctx, &model.User{ID: uuid.New(), Status: model.UserStatusActive, IsAdmin: true})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected profile-only update to succeed without token invalidation, got status %d body %s", rec.Code, rec.Body.String())
	}
	if store.updatedID != targetID {
		t.Fatalf("expected user %s to be updated, got %s", targetID, store.updatedID)
	}
}

func TestAdminRunSyncRejectsWhenSyncDisabled(t *testing.T) {
	starter := &fakeSyncStarter{}
	handler := NewAdminHandler(nil, nil, nil, nil, starter, nil, false)

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
	handler := NewAdminHandler(nil, nil, nil, nil, nil, nil, true)

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
	handler := NewAdminHandler(nil, nil, nil, nil, starter, nil, true)
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
