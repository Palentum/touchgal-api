package publicapi

import (
	"context"
	"testing"

	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeGameStore struct {
	searchAllowNsfw    bool
	detailAllowNsfw    bool
	detailErr          error
	resourcesCalled    bool
	resourcesUniqueID  string
	resourcesSiteURL   string
	resourcesAllowNsfw bool
	resourcesErr       error
	patchesCalled      bool
	patchesUniqueID    string
	patchesSiteURL     string
	patchesAllowNsfw   bool
	patchesErr         error
}

func (f *fakeGameStore) Search(ctx context.Context, keyword string, page, limit int, allowNsfw bool) (model.GameSearchResult, error) {
	f.searchAllowNsfw = allowNsfw
	return model.GameSearchResult{Items: []model.GameSearchItem{{Name: "Summer", UniqueID: "abcd1234"}}}, nil
}
func (f *fakeGameStore) Detail(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (*model.GameDetail, error) {
	f.detailAllowNsfw = allowNsfw
	if f.detailErr != nil {
		return nil, f.detailErr
	}
	return nil, model.ErrNotFound
}

func (f *fakeGameStore) Resources(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (model.GameResourceList, error) {
	f.resourcesCalled = true
	f.resourcesUniqueID = uniqueID
	f.resourcesSiteURL = touchgalSiteURL
	f.resourcesAllowNsfw = allowNsfw
	if f.resourcesErr != nil {
		return model.GameResourceList{}, f.resourcesErr
	}
	return model.GameResourceList{Items: []model.GameResourceItem{}}, nil
}

func (f *fakeGameStore) Patches(ctx context.Context, uniqueID, touchgalSiteURL string, allowNsfw bool) (model.GameResourceList, error) {
	f.patchesCalled = true
	f.patchesUniqueID = uniqueID
	f.patchesSiteURL = touchgalSiteURL
	f.patchesAllowNsfw = allowNsfw
	if f.patchesErr != nil {
		return model.GameResourceList{}, f.patchesErr
	}
	return model.GameResourceList{Items: []model.GameResourceItem{}}, nil
}

func TestSearchParamsValidation(t *testing.T) {
	keyword, page, limit, err := NormalizeSearch(" summer ", 0, 999)
	if err != nil || keyword != "summer" || page != 1 || limit != maxSearchLimit {
		t.Fatalf("unexpected normalization: %q %d %d %v", keyword, page, limit, err)
	}
	if _, _, _, err := NormalizeSearch("", 1, 10); err != model.ErrInvalidInput {
		t.Fatal("empty keyword accepted")
	}
	if _, _, _, err := NormalizeSearch("ab", 1, 10); err != model.ErrInvalidInput {
		t.Fatal("short keyword accepted")
	}
	if _, _, _, err := NormalizeSearch("あい", 1, 10); err != model.ErrInvalidInput {
		t.Fatal("short unicode keyword accepted")
	}
	if _, _, _, err := NormalizeSearch("summer", maxSearchPage+1, 10); err != model.ErrInvalidInput {
		t.Fatal("page above cap accepted")
	}
}

func TestParseAllowNsfw(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
		err  error
	}{
		{raw: "", want: false},
		{raw: "true", want: true},
		{raw: "false", want: false},
		{raw: "1", err: model.ErrInvalidInput},
		{raw: "0", err: model.ErrInvalidInput},
		{raw: "t", err: model.ErrInvalidInput},
		{raw: "TRUE", err: model.ErrInvalidInput},
		{raw: "yes", err: model.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseAllowNsfw(tt.raw)
			if err != tt.err {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
func TestSearchParamsRejectInvalidUTF8(t *testing.T) {
	if _, _, _, err := NormalizeSearch(string([]byte{0xff, 0xff, 0xff}), 1, 10); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid UTF-8 rejection, got %v", err)
	}
}

func TestDetailNotFound(t *testing.T) {
	svc := NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, &fakeGameStore{})
	if _, err := svc.Detail(context.Background(), "abcd1234", false); err != model.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServicePassesAllowNsfw(t *testing.T) {
	store := &fakeGameStore{detailErr: model.ErrNotFound}
	svc := NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, store)

	if _, err := svc.Search(context.Background(), "summer", 1, 20, true); err != nil {
		t.Fatalf("search: %v", err)
	}
	if !store.searchAllowNsfw {
		t.Fatal("expected Search to pass allowNsfw=true")
	}
	if _, err := svc.Detail(context.Background(), "abcd1234", true); err != model.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	if !store.detailAllowNsfw {
		t.Fatal("expected Detail to pass allowNsfw=true")
	}
}

func TestResourcesPassesAllowNsfwAndSiteURL(t *testing.T) {
	store := &fakeGameStore{}
	svc := NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, store)

	if _, err := svc.Resources(context.Background(), "abcd1234", true); err != nil {
		t.Fatalf("resources: %v", err)
	}
	if !store.resourcesCalled {
		t.Fatal("expected Resources store call")
	}
	if store.resourcesUniqueID != "abcd1234" {
		t.Fatalf("unexpected uniqueId: %q", store.resourcesUniqueID)
	}
	if store.resourcesSiteURL != "https://www.touchgal.ink" {
		t.Fatalf("unexpected TouchGal site URL: %q", store.resourcesSiteURL)
	}
	if !store.resourcesAllowNsfw {
		t.Fatal("expected Resources to pass allowNsfw=true")
	}
}

func TestPatchesRejectsInvalidUniqueIDBeforeStore(t *testing.T) {
	store := &fakeGameStore{}
	svc := NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, store)

	if _, err := svc.Patches(context.Background(), "bad", false); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if store.patchesCalled {
		t.Fatal("patches store should not be called for invalid uniqueId")
	}
}

func TestResourcesReturnsStoreNotFound(t *testing.T) {
	store := &fakeGameStore{resourcesErr: model.ErrNotFound}
	svc := NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, store)

	if _, err := svc.Resources(context.Background(), "abcd1234", false); err != model.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestUniqueIDValidation(t *testing.T) {
	if err := ValidateUniqueID("abcd1234"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUniqueID("abcd12345"); err != model.ErrInvalidInput {
		t.Fatal("invalid uniqueId accepted")
	}
}
