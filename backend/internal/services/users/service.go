package users

import (
	"context"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

const (
	maxSearchQueryLength  = 160
	maxDisplayNameLength  = 80
	maxEmailLength        = 254
	defaultAdminListLimit = 20
	maxAdminListPage      = 100
	maxAdminListLimit     = 100
)

type AdminUpdate struct {
	Email       *string
	DisplayName *string
	Status      *string
	MinuteLimit *int
	DailyLimit  *int
}

type Store interface {
	ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error)
	UpdateAdmin(ctx context.Context, id uuid.UUID, email, displayName, status *string, minuteLimit, dailyLimit *int) (*model.User, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
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
	page, limit, err := normalizePage(page, limit)
	if err != nil {
		return nil, err
	}
	return s.store.ListAdmin(ctx, status, query, page, limit)
}

func (s *Service) UpdateAdmin(ctx context.Context, actorID, id uuid.UUID, input AdminUpdate) (*model.User, error) {
	if input.Email == nil && input.DisplayName == nil && input.Status == nil && input.MinuteLimit == nil && input.DailyLimit == nil {
		return nil, model.ErrInvalidInput
	}
	if input.Email != nil {
		email, err := normalizeEmail(*input.Email)
		if err != nil {
			return nil, err
		}
		input.Email = &email
	}
	if input.DisplayName != nil {
		displayName := strings.TrimSpace(*input.DisplayName)
		if len(displayName) > maxDisplayNameLength {
			return nil, model.ErrInvalidInput
		}
		input.DisplayName = &displayName
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !validStatus(status) {
			return nil, model.ErrInvalidInput
		}
		if actorID == id && status != model.UserStatusActive {
			return nil, model.ErrInvalidInput
		}
		input.Status = &status
	}
	if (input.MinuteLimit == nil) != (input.DailyLimit == nil) {
		return nil, model.ErrInvalidInput
	}
	if input.MinuteLimit != nil {
		if *input.MinuteLimit <= 0 || *input.DailyLimit <= 0 || *input.DailyLimit < *input.MinuteLimit {
			return nil, model.ErrInvalidInput
		}
	}
	return s.store.UpdateAdmin(ctx, id, input.Email, input.DisplayName, input.Status, input.MinuteLimit, input.DailyLimit)
}

func (s *Service) DeleteAdmin(ctx context.Context, actorID, id uuid.UUID) error {
	if actorID == id {
		return model.ErrInvalidInput
	}
	return s.store.DeleteByID(ctx, id)
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > maxEmailLength {
		return "", model.ErrInvalidInput
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", model.ErrInvalidInput
	}
	return email, nil
}

func validStatus(status string) bool {
	return status == model.UserStatusActive || status == model.UserStatusDisabled
}

func normalizePage(page, limit int) (int, int, error) {
	if page < 1 {
		page = 1
	}
	if page > maxAdminListPage {
		return 0, 0, model.ErrInvalidInput
	}
	if limit < 1 {
		limit = defaultAdminListLimit
	}
	if limit > maxAdminListLimit {
		limit = maxAdminListLimit
	}
	return page, limit, nil
}
