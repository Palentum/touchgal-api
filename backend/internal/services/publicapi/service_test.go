package publicapi

import (
	"context"
	"testing"

	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeGameStore struct{}

func (fakeGameStore) Search(ctx context.Context, keyword string, page, limit int) (model.GameSearchResult, error) {
	return model.GameSearchResult{Items: []model.GameSearchItem{{Name: "Summer", UniqueID: "abcd1234"}}}, nil
}
func (fakeGameStore) Detail(ctx context.Context, uniqueID, touchgalSiteURL string) (*model.GameDetail, error) {
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
func TestSearchParamsRejectInvalidUTF8(t *testing.T) {
	if _, _, _, err := NormalizeSearch(string([]byte{0xff, 0xff, 0xff}), 1, 10); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid UTF-8 rejection, got %v", err)
	}
}

func TestDetailNotFound(t *testing.T) {
	svc := NewService(config.Config{TouchGalSiteURL: "https://www.touchgal.ink"}, fakeGameStore{})
	if _, err := svc.Detail(context.Background(), "abcd1234"); err != model.ErrNotFound {
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
