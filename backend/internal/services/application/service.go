package application

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/services/email"
)

type Store interface {
	Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput, minuteLimit, dailyLimit int) (*model.Application, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Application, error)
	ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminApplication, error)
	UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status string, minuteLimit, dailyLimit int) (*model.Application, error)
}

type UserStore interface {
	ListActiveAdminEmails(ctx context.Context) ([]string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type Service struct {
	store  Store
	users  UserStore
	cfg    config.Config
	mailer email.Mailer
	logger zerolog.Logger
}

func NewService(cfg config.Config, store Store, users UserStore, mailer email.Mailer, logger zerolog.Logger) *Service {
	return &Service{cfg: cfg, store: store, users: users, mailer: mailer, logger: logger}
}

const (
	defaultAdminListLimit = 20
	maxAdminListPage      = 100
	maxAdminListLimit     = 100
)

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
	app, err := s.store.Create(ctx, userID, input, s.cfg.DefaultTokenMinuteLimit, s.cfg.DefaultTokenDailyLimit)
	if err != nil {
		return nil, err
	}
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.MailSendTimeout())
	defer cancel()
	s.notifyApplicationSubmitted(notifyCtx, *app)
	return app, nil
}

func adminApplicationsURL(publicURL string) string {
	base := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if base == "" {
		return "/admin/applications"
	}
	return base + "/admin/applications"
}

func dashboardTokensURL(publicURL string) string {
	base := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if base == "" {
		return "/dashboard/tokens"
	}
	return base + "/dashboard/tokens"
}

func (s *Service) notifyApplicationSubmitted(ctx context.Context, app model.Application) {
	if s.users == nil || s.mailer == nil {
		return
	}
	emails, err := s.users.ListActiveAdminEmails(ctx)
	if err != nil {
		s.logger.Error().Err(err).Str("application_id", app.ID.String()).Msg("list active admin email recipients failed")
		return
	}
	if len(emails) == 0 {
		s.logger.Warn().Str("application_id", app.ID.String()).Msg("no active admin email recipients for application notification")
		return
	}
	if err := s.mailer.SendApplicationSubmitted(emails, app, adminApplicationsURL(s.cfg.PublicURL)); err != nil {
		s.logger.Error().Err(err).Str("application_id", app.ID.String()).Int("recipient_count", len(emails)).Msg("send application submitted admin notification failed")
	}
}

func (s *Service) notifyApplicationApproved(ctx context.Context, app model.Application) {
	if s.users == nil || s.mailer == nil {
		return
	}
	user, err := s.users.GetByID(ctx, app.UserID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("application_id", app.ID.String()).
			Str("user_id", app.UserID.String()).
			Msg("get application applicant email failed")
		return
	}
	if strings.TrimSpace(user.Email) == "" {
		s.logger.Warn().
			Str("application_id", app.ID.String()).
			Str("user_id", app.UserID.String()).
			Msg("application applicant has no email for approval notification")
		return
	}
	if err := s.mailer.SendApplicationApproved(user.Email, app, dashboardTokensURL(s.cfg.PublicURL)); err != nil {
		s.logger.Error().
			Err(err).
			Str("application_id", app.ID.String()).
			Str("user_id", app.UserID.String()).
			Str("email", user.Email).
			Msg("send application approved user notification failed")
	}
}

func (s *Service) ListMine(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	return s.store.ListByUser(ctx, userID)
}

func (s *Service) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminApplication, error) {
	page, limit, err := normalizePage(page, limit)
	if err != nil {
		return nil, err
	}
	return s.store.ListAdmin(ctx, status, page, limit)
}

func (s *Service) Review(ctx context.Context, id, adminID uuid.UUID, status string, minuteLimit, dailyLimit int) (*model.Application, error) {
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
	app, err := s.store.UpdateReview(ctx, id, adminID, status, minuteLimit, dailyLimit)
	if err != nil {
		return nil, err
	}
	if status == model.ApplicationApproved {
		notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.MailSendTimeout())
		defer cancel()
		s.notifyApplicationApproved(notifyCtx, *app)
	}
	return app, nil
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
