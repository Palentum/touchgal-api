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
	createErr         error
	created           bool
	listStatus        string
	listPage          int
	listLimit         int
	reviewApp         *model.Application
	reviewErr         error
	reviewStatus      string
	reviewMinuteLimit int
	reviewDailyLimit  int
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
func (f *fakeApplicationStore) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminApplication, error) {
	f.listStatus = status
	f.listPage = page
	f.listLimit = limit
	return nil, nil
}
func (f *fakeApplicationStore) UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status string, minuteLimit, dailyLimit int) (*model.Application, error) {
	f.reviewStatus = status
	f.reviewMinuteLimit = minuteLimit
	f.reviewDailyLimit = dailyLimit
	if f.reviewErr != nil {
		return nil, f.reviewErr
	}
	if f.reviewApp != nil {
		return f.reviewApp, nil
	}
	return &model.Application{
		ID:                 id,
		Status:             status,
		DefaultMinuteLimit: minuteLimit,
		DefaultDailyLimit:  dailyLimit,
	}, nil
}

type fakeApplicationUserStore struct {
	adminEmails []string
	adminErr    error
	adminCalled bool
	user        *model.User
	getErr      error
	getCalled   bool
	getID       uuid.UUID
}

func (f *fakeApplicationUserStore) ListActiveAdminEmails(ctx context.Context) ([]string, error) {
	f.adminCalled = true
	if f.adminErr != nil {
		return nil, f.adminErr
	}
	return f.adminEmails, nil
}

func (f *fakeApplicationUserStore) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	f.getCalled = true
	f.getID = id
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.user == nil {
		return nil, model.ErrNotFound
	}
	return f.user, nil
}

type fakeApplicationMailer struct {
	submittedTo        []string
	submittedApp       model.Application
	submittedReviewURL string
	submittedErr       error
	submittedCalled    bool
	approvedCalled     bool
	approvedTo         string
	approvedApp        model.Application
	approvedURL        string
	approvedErr        error
}

func (f *fakeApplicationMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	return nil
}

func (f *fakeApplicationMailer) SendApplicationSubmitted(to []string, app model.Application, reviewURL string) error {
	f.submittedCalled = true
	f.submittedTo = to
	f.submittedApp = app
	f.submittedReviewURL = reviewURL
	return f.submittedErr
}

func (f *fakeApplicationMailer) SendApplicationApproved(to string, app model.Application, dashboardURL string) error {
	f.approvedCalled = true
	f.approvedTo = to
	f.approvedApp = app
	f.approvedURL = dashboardURL
	return f.approvedErr
}

func TestApplicationAlreadySubmitted(t *testing.T) {
	store := &fakeApplicationStore{createErr: model.ErrApplicationExists}
	admins := &fakeApplicationUserStore{adminEmails: []string{"admin@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store, admins, mailer, zerolog.Nop())
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != model.ErrApplicationExists {
		t.Fatalf("expected ErrApplicationExists, got %v", err)
	}
	if !store.created {
		t.Fatal("expected create path to be called")
	}
	if admins.adminCalled || mailer.submittedCalled {
		t.Fatal("failed application create must not notify admins")
	}
}

func TestCreateApplicationUsesDefaultLimits(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store, &fakeApplicationUserStore{}, &fakeApplicationMailer{}, zerolog.Nop())
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
	admins := &fakeApplicationUserStore{adminEmails: []string{"admin@example.com", "ops@example.com"}}
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
	if !admins.adminCalled {
		t.Fatal("active admin recipients were not queried")
	}
	if !mailer.submittedCalled {
		t.Fatal("application notification was not sent")
	}
	if !slices.Equal(mailer.submittedTo, []string{"admin@example.com", "ops@example.com"}) {
		t.Fatalf("unexpected recipients: %#v", mailer.submittedTo)
	}
	if mailer.submittedApp.ID != app.ID {
		t.Fatalf("notification app ID mismatch: sent=%s created=%s", mailer.submittedApp.ID, app.ID)
	}
	if mailer.submittedReviewURL != "https://portal.example.com/admin/applications" {
		t.Fatalf("unexpected review URL: %q", mailer.submittedReviewURL)
	}
}

func TestCreateApplicationIgnoresNotificationFailures(t *testing.T) {
	store := &fakeApplicationStore{}
	admins := &fakeApplicationUserStore{adminEmails: []string{"admin@example.com"}}
	mailer := &fakeApplicationMailer{submittedErr: errors.New("smtp down")}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store, admins, mailer, zerolog.Nop())

	app, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != nil {
		t.Fatalf("notification failure must not fail create: %v", err)
	}
	if app == nil {
		t.Fatal("expected created application")
	}
	if !mailer.submittedCalled {
		t.Fatal("expected notification attempt")
	}
}

func TestCreateApplicationInvalidInputDoesNotNotifyAdmins(t *testing.T) {
	store := &fakeApplicationStore{}
	admins := &fakeApplicationUserStore{adminEmails: []string{"admin@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{}, store, admins, mailer, zerolog.Nop())

	_, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != model.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if store.created {
		t.Fatal("invalid input must not reach the store")
	}
	if admins.adminCalled || mailer.submittedCalled {
		t.Fatal("invalid input must not notify admins")
	}
}

func TestReviewApplicationApprovedNotifiesApplicant(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()
	adminID := uuid.New()
	store := &fakeApplicationStore{
		reviewApp: &model.Application{
			ID:                 appID,
			UserID:             userID,
			Status:             model.ApplicationApproved,
			DefaultMinuteLimit: 10,
			DefaultDailyLimit:  100,
		},
	}
	users := &fakeApplicationUserStore{user: &model.User{ID: userID, Email: "dev@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{PublicURL: "https://portal.example.com/"}, store, users, mailer, zerolog.Nop())

	app, err := svc.Review(context.Background(), appID, adminID, model.ApplicationApproved, 10, 100)
	if err != nil {
		t.Fatalf("review application failed: %v", err)
	}
	if app == nil || app.ID != appID {
		t.Fatalf("unexpected reviewed application: %+v", app)
	}
	if !users.getCalled || users.getID != userID {
		t.Fatalf("applicant lookup mismatch: called=%t id=%s", users.getCalled, users.getID)
	}
	if !mailer.approvedCalled {
		t.Fatal("approved application notification was not sent")
	}
	if mailer.approvedTo != "dev@example.com" {
		t.Fatalf("unexpected approved notification recipient: %q", mailer.approvedTo)
	}
	if mailer.approvedApp.ID != appID {
		t.Fatalf("approved notification app ID mismatch: %s", mailer.approvedApp.ID)
	}
	if mailer.approvedURL != "https://portal.example.com/dashboard/tokens" {
		t.Fatalf("unexpected dashboard URL: %q", mailer.approvedURL)
	}
}

func TestReviewApplicationApprovedIgnoresNotificationFailures(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()
	store := &fakeApplicationStore{
		reviewApp: &model.Application{
			ID:                 appID,
			UserID:             userID,
			Status:             model.ApplicationApproved,
			DefaultMinuteLimit: 10,
			DefaultDailyLimit:  100,
		},
	}
	users := &fakeApplicationUserStore{user: &model.User{ID: userID, Email: "dev@example.com"}}
	mailer := &fakeApplicationMailer{approvedErr: errors.New("smtp down")}
	svc := NewService(config.Config{}, store, users, mailer, zerolog.Nop())

	app, err := svc.Review(context.Background(), appID, uuid.New(), model.ApplicationApproved, 10, 100)
	if err != nil {
		t.Fatalf("notification failure must not fail review: %v", err)
	}
	if app == nil {
		t.Fatal("expected reviewed application")
	}
}

func TestReviewApplicationRejectedDoesNotNotifyApplicant(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()
	store := &fakeApplicationStore{
		reviewApp: &model.Application{
			ID:                 appID,
			UserID:             userID,
			Status:             model.ApplicationRejected,
			DefaultMinuteLimit: 10,
			DefaultDailyLimit:  100,
		},
	}
	users := &fakeApplicationUserStore{user: &model.User{ID: userID, Email: "dev@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{}, store, users, mailer, zerolog.Nop())

	if _, err := svc.Review(context.Background(), appID, uuid.New(), model.ApplicationRejected, 10, 100); err != nil {
		t.Fatalf("review application failed: %v", err)
	}
	if users.getCalled {
		t.Fatal("rejected application must not query applicant email")
	}
	if mailer.approvedCalled {
		t.Fatal("rejected application must not send approved notification")
	}
}

func TestReviewApplicationStoreErrorDoesNotNotifyApplicant(t *testing.T) {
	store := &fakeApplicationStore{reviewErr: model.ErrNotFound}
	users := &fakeApplicationUserStore{user: &model.User{ID: uuid.New(), Email: "dev@example.com"}}
	mailer := &fakeApplicationMailer{}
	svc := NewService(config.Config{}, store, users, mailer, zerolog.Nop())

	if _, err := svc.Review(context.Background(), uuid.New(), uuid.New(), model.ApplicationApproved, 10, 100); err != model.ErrNotFound {
		t.Fatalf("expected store error, got %v", err)
	}
	if users.getCalled {
		t.Fatal("failed review must not query applicant email")
	}
	if mailer.approvedCalled {
		t.Fatal("failed review must not send approved notification")
	}
}

func TestListAdminNormalizesPageAndLimit(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{}, store, &fakeApplicationUserStore{}, &fakeApplicationMailer{}, zerolog.Nop())
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
	svc := NewService(config.Config{}, store, &fakeApplicationUserStore{}, &fakeApplicationMailer{}, zerolog.Nop())
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
