package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestRequireUserRejectsDisabledAccount(t *testing.T) {
	called := false
	handler := RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	req = req.WithContext(WithUser(req.Context(), &model.User{ID: uuid.New(), Status: model.UserStatusDisabled}))
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if called {
		t.Fatal("next handler must not run for disabled account")
	}
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "ACCOUNT_DISABLED") {
		t.Fatalf("expected ACCOUNT_DISABLED response, got %s", res.Body.String())
	}
}

func TestRequireAdminRejectsDisabledAccountBeforeAdminCheck(t *testing.T) {
	handler := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not run for disabled account")
	}))
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req = req.WithContext(WithUser(req.Context(), &model.User{ID: uuid.New(), Status: model.UserStatusDisabled, IsAdmin: false}))
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "ACCOUNT_DISABLED") {
		t.Fatalf("expected ACCOUNT_DISABLED response, got %s", res.Body.String())
	}
}
