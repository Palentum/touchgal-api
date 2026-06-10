package publicapi

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type GameStore interface {
	Search(ctx context.Context, keyword string, page, limit int) (model.GameSearchResult, error)
	Detail(ctx context.Context, uniqueID, touchgalSiteURL string) (*model.GameDetail, error)
}

type Service struct {
	cfg   config.Config
	games GameStore
}

func NewService(cfg config.Config, games GameStore) *Service { return &Service{cfg: cfg, games: games} }

const (
	minSearchKeywordRunes = 3
	maxSearchKeywordRunes = 100
	maxSearchPage         = 100
	defaultSearchLimit    = 20
	maxSearchLimit        = 50
)

func NormalizeSearch(keyword string, page, limit int) (string, int, int, error) {
	keyword = strings.TrimSpace(keyword)
	if !utf8.ValidString(keyword) {
		return "", 0, 0, model.ErrInvalidInput
	}
	keywordRunes := utf8.RuneCountInString(keyword)
	if keywordRunes < minSearchKeywordRunes || keywordRunes > maxSearchKeywordRunes {
		return "", 0, 0, model.ErrInvalidInput
	}
	if page < 1 {
		page = 1
	}
	if page > maxSearchPage {
		return "", 0, 0, model.ErrInvalidInput
	}
	if limit < 1 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	return keyword, page, limit, nil
}

func ValidateUniqueID(uniqueID string) error {
	if len(uniqueID) != 8 {
		return model.ErrInvalidInput
	}
	for _, ch := range uniqueID {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return model.ErrInvalidInput
		}
	}
	return nil
}

func (s *Service) Search(ctx context.Context, keyword string, page, limit int) (model.GameSearchResult, error) {
	keyword, page, limit, err := NormalizeSearch(keyword, page, limit)
	if err != nil {
		return model.GameSearchResult{}, err
	}
	return s.games.Search(ctx, keyword, page, limit)
}

func (s *Service) Detail(ctx context.Context, uniqueID string) (*model.GameDetail, error) {
	if err := ValidateUniqueID(uniqueID); err != nil {
		return nil, err
	}
	return s.games.Detail(ctx, uniqueID, s.cfg.TouchGalSiteURL)
}
