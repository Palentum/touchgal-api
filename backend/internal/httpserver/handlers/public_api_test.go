package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/services/publicapi"
)

type publicAPIFakeGameStore struct {
	searchAllowNsfw bool
	detailCalled    bool
}

func (f *publicAPIFakeGameStore) Search(ctx context.Context, keyword string, page, limit int, allowNsfw bool) (model.GameSearchResult, error) {
	f.searchAllowNsfw = allowNsfw
	return model.GameSearchResult{Items: []model.GameSearchItem{{Name: "Summer", UniqueID: "abcd1234"}}}, nil
}

func (f *publicAPIFakeGameStore) Detail(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (*model.GameDetail, error) {
	f.detailCalled = true
	return &model.GameDetail{UniqueID: uniqueID, Name: "Summer"}, nil
}

func TestPublicAPISearchPassesAllowNsfw(t *testing.T) {
	store := &publicAPIFakeGameStore{}
	handler := NewPublicAPIHandler(publicapi.NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, store))
	req := httptest.NewRequest(http.MethodGet, "/v1/games/search?keyword=summer&allowNsfw=true", nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !store.searchAllowNsfw {
		t.Fatal("expected search store to receive allowNsfw=true")
	}
}

func TestPublicAPIDetailRejectsNumericAllowNsfw(t *testing.T) {
	store := &publicAPIFakeGameStore{}
	handler := NewPublicAPIHandler(publicapi.NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, store))
	req := httptest.NewRequest(http.MethodGet, "/v1/games/abcd1234?allowNsfw=1", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("uniqueId", "abcd1234")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()

	handler.Detail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"BAD_REQUEST"`) {
		t.Fatalf("expected BAD_REQUEST response, got %s", rec.Body.String())
	}
	if store.detailCalled {
		t.Fatal("detail store should not be called for invalid allowNsfw")
	}
}
