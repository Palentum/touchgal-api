package application

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeApplicationStore struct {
	createErr  error
	created    bool
	listStatus string
	listPage   int
	listLimit  int
}

func (f *fakeApplicationStore) Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput, minuteLimit, dailyLimit int) (*model.Application, error) {
	f.created = true
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &model.Application{
		ID:                    uuid.New(),
		UserID:                userID,
		ApplicantName:         input.ApplicantName,
		ProjectName:           input.ProjectName,
		ProjectURL:            input.ProjectURL,
		ExpectedDailyRequests: input.ExpectedDailyRequests,
		UsageScenario:         input.UsageScenario,
		Status:                model.ApplicationPending,
		DefaultMinuteLimit:    minuteLimit,
		DefaultDailyLimit:     dailyLimit,
	}, nil
}
func (f *fakeApplicationStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	return nil, nil
}
func (f *fakeApplicationStore) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.Application, error) {
	f.listStatus = status
	f.listPage = page
	f.listLimit = limit
	return nil, nil
}
func (f *fakeApplicationStore) UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status string, minuteLimit, dailyLimit int) (*model.Application, error) {
	return nil, nil
}

type fakeAdminRecipientStore struct {
	emails []string
	err    error
	called bool
}

func (f *fakeAdminRecipientStore) ListActiveAdminEmails(ctx context.Context) ([]string, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return f.emails, nil
}

type fakeApplicationMailer struct {
	sentTo    []string
	sentApp   model.Application
	reviewURL string
	err       error
	called    bool
}

func (f *fakeApplicationMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	return nil
}

func (f *fakeApplicationMailer) SendApplicationSubmitted(to []string, app model.Application, reviewURL string) error {
	f.called = true
	f.sentTo = to
	f.sentApp = app
	f.reviewURL = reviewURL
	return f.err
}

func TestApplicationAlreadySubmitted(t *testing.T) {
	store := &fakeApplicationStore{createErr: model.ErrApplicationExists}
	admins := &fakeAdminRecipientStore{emails: []string{"admin@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store, admins, mailer, zerolog.Nop())
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != model.ErrApplicationExists {
		t.Fatalf("expected ErrApplicationExists, got %v", err)
	}
	if !store.created {
		t.Fatal("expected create path to be called")
	}
	if admins.called || mailer.called {
		t.Fatal("failed application create must not notify admins")
	}
}

func TestCreateApplicationUsesDefaultLimits(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store, &fakeAdminRecipientStore{}, &fakeApplicationMailer{}, zerolog.Nop())
	app, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != nil {
		t.Fatalf("create application failed: %v", err)
	}
	if app.Status != model.ApplicationPending || app.DefaultMinuteLimit != 60 || app.DefaultDailyLimit != 5000 {
		t.Fatalf("unexpected created application: %+v", app)
	}
}

func TestCreateApplicationNotifiesActiveAdmins(t *testing.T) {
	store := &fakeApplicationStore{}
	admins := &fakeAdminRecipientStore{emails: []string{"admin@example.com", "ops@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(
		config.Config{
			PublicURL:               "https://portal.example.com/",
			DefaultTokenMinuteLimit: 60,
			DefaultTokenDailyLimit:  5000,
		},
		store,
		admins,
		mailer,
		zerolog.Nop(),
	)

	app, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{
		ApplicantName:         "Kun",
		ProjectName:           "Docs Bot",
		ProjectURL:            "https://example.com",
		ExpectedDailyRequests: 100,
		UsageScenario:         "展示条目信息",
	})
	if err != nil {
		t.Fatalf("create application failed: %v", err)
	}
	if !admins.called {
		t.Fatal("active admin recipients were not queried")
	}
	if !mailer.called {
		t.Fatal("application notification was not sent")
	}
	if !slices.Equal(mailer.sentTo, []string{"admin@example.com", "ops@example.com"}) {
		t.Fatalf("unexpected recipients: %#v", mailer.sentTo)
	}
	if mailer.sentApp.ID != app.ID {
		t.Fatalf("notification app ID mismatch: sent=%s created=%s", mailer.sentApp.ID, app.ID)
	}
	if mailer.reviewURL != "https://portal.example.com/admin/applications" {
		t.Fatalf("unexpected review URL: %q", mailer.reviewURL)
	}
}

func TestCreateApplicationIgnoresNotificationFailures(t *testing.T) {
	store := &fakeApplicationStore{}
	admins := &fakeAdminRecipientStore{emails: []string{"admin@example.com"}}
	mailer := &fakeApplicationMailer{err: errors.New("smtp down")}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store, admins, mailer, zerolog.Nop())

	app, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != nil {
		t.Fatalf("notification failure must not fail create: %v", err)
	}
	if app == nil {
		t.Fatal("expected created application")
	}
	if !mailer.called {
		t.Fatal("expected notification attempt")
	}
}

func TestCreateApplicationInvalidInputDoesNotNotifyAdmins(t *testing.T) {
	store := &fakeApplicationStore{}
	admins := &fakeAdminRecipientStore{emails: []string{"admin@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{}, store, admins, mailer, zerolog.Nop())

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != model.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if store.created {
		t.Fatal("invalid input must not reach the store")
	}
	if admins.called || mailer.called {
		t.Fatal("invalid input must not notify admins")
	}
}

func TestListAdminNormalizesPageAndLimit(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{}, store, &fakeAdminRecipientStore{}, &fakeApplicationMailer{}, zerolog.Nop())
	if _, err := svc.ListAdmin(context.Background(), model.ApplicationPending, 0, 150); err != nil {
		t.Fatalf("ListAdmin returned error: %v", err)
	}
	if store.listStatus != model.ApplicationPending {
		t.Fatalf("unexpected status filter: %q", store.listStatus)
	}
	if store.listPage != 1 || store.listLimit != maxAdminListLimit {
		t.Fatalf("unexpected pagination: page=%d limit=%d", store.listPage, store.listLimit)
	}
}

func TestListAdminRejectsPageAboveCap(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{}, store, &fakeAdminRecipientStore{}, &fakeApplicationMailer{}, zerolog.Nop())
	if _, err := svc.ListAdmin(context.Background(), "", maxAdminListPage+1, 20); err != model.ErrInvalidInput {
		t.Fatalf("expected page cap error, got %v", err)
	}
	if store.listPage != 0 || store.listLimit != 0 {
		t.Fatal("invalid admin pagination must not reach the store")
	}
}

func TestValidateApplicationInput(t *testing.T) {
	input := model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 100, UsageScenario: "用于展示条目信息"}
	if err := ValidateInput(&input); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	exactDashboardPayload := model.CreateApplicationInput{ApplicantName: "测试负责人", ProjectName: "测试项目", ProjectURL: "https://test.com", ExpectedDailyRequests: 1234, UsageScenario: "测试使用场景"}
	if err := ValidateInput(&exactDashboardPayload); err != nil {
		t.Fatalf("dashboard apply payload rejected: %v", err)
	}
	bad := input
	bad.ProjectURL = "javascript:alert(1)"
	if err := ValidateInput(&bad); err == nil {
		t.Fatal("invalid URL accepted")
	}
}
