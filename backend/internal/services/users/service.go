package users

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

const maxSearchQueryLength = 160

type Store interface {
	ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.User, error)
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error) {
	status = strings.TrimSpace(status)
	if status != "" && !validStatus(status) {
		return nil, model.ErrInvalidInput
	}
	query = strings.TrimSpace(query)
	if len(query) > maxSearchQueryLength {
		return nil, model.ErrInvalidInput
	}
	page, limit = normalizePage(page, limit, 100)
	return s.store.ListAdmin(ctx, status, query, page, limit)
}

func (s *Service) UpdateStatus(ctx context.Context, actorID, id uuid.UUID, status string) (*model.User, error) {
	status = strings.TrimSpace(status)
	if !validStatus(status) {
		return nil, model.ErrInvalidInput
	}
	if actorID == id && status != model.UserStatusActive {
		return nil, model.ErrInvalidInput
	}
	return s.store.UpdateStatus(ctx, id, status)
}

func validStatus(status string) bool {
	return status == model.UserStatusActive || status == model.UserStatusDisabled
}

func normalizePage(page, limit, maxLimit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}
