package publicapi

import (
	"context"
	"testing"

	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeGameStore struct {
	searchAllowNsfw bool
	detailAllowNsfw bool
	detailErr       error
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

func TestUniqueIDValidation(t *testing.T) {
	if err := ValidateUniqueID("abcd1234"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUniqueID("abcd12345"); err != model.ErrInvalidInput {
		t.Fatal("invalid uniqueId accepted")
	}
}
