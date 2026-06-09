package application

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type Store interface {
	Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput, minuteLimit, dailyLimit int) (*model.Application, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Application, error)
	ListAdmin(ctx context.Context, status string, page, limit int) ([]model.Application, error)
	UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status, note string, minuteLimit, dailyLimit int) (*model.Application, error)
}

type Service struct {
	store Store
	cfg   config.Config
}

func NewService(cfg config.Config, store Store) *Service { return &Service{cfg: cfg, store: store} }

func ValidateInput(input *model.CreateApplicationInput) error {
	input.ApplicantName = strings.TrimSpace(input.ApplicantName)
	input.ProjectName = strings.TrimSpace(input.ProjectName)
	input.ProjectURL = strings.TrimSpace(input.ProjectURL)
	input.UsageScenario = strings.TrimSpace(input.UsageScenario)
	if input.ApplicantName == "" || len(input.ApplicantName) > 100 {
		return model.ErrInvalidInput
	}
	if len(input.ProjectName) > 160 {
		return model.ErrInvalidInput
	}
	if input.ProjectURL == "" || len(input.ProjectURL) > 1000 {
		return model.ErrInvalidInput
	}
	parsed, err := url.ParseRequestURI(input.ProjectURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return model.ErrInvalidInput
	}
	if input.ExpectedDailyRequests <= 0 {
		return model.ErrInvalidInput
	}
	if input.UsageScenario == "" {
		return model.ErrInvalidInput
	}
	return nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput) (*model.Application, error) {
	if err := ValidateInput(&input); err != nil {
		return nil, err
	}
	return s.store.Create(ctx, userID, input, s.cfg.DefaultTokenMinuteLimit, s.cfg.DefaultTokenDailyLimit)
}

func (s *Service) ListMine(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	return s.store.ListByUser(ctx, userID)
}

func (s *Service) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.Application, error) {
	page, limit = normalizePage(page, limit, 100)
	return s.store.ListAdmin(ctx, status, page, limit)
}

func (s *Service) Review(ctx context.Context, id, adminID uuid.UUID, status, note string, minuteLimit, dailyLimit int) (*model.Application, error) {
	if status != model.ApplicationApproved && status != model.ApplicationRejected && status != model.ApplicationRevoked {
		return nil, model.ErrInvalidInput
	}
	if minuteLimit <= 0 {
		minuteLimit = s.cfg.DefaultTokenMinuteLimit
	}
	if dailyLimit <= 0 {
		dailyLimit = s.cfg.DefaultTokenDailyLimit
	}
	if dailyLimit < minuteLimit {
		return nil, model.ErrInvalidInput
	}
	return s.store.UpdateReview(ctx, id, adminID, status, strings.TrimSpace(note), minuteLimit, dailyLimit)
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
