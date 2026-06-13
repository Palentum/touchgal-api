package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
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
	return &model.Application{ID: uuid.New(), UserID: userID, Status: model.ApplicationPending, DefaultMinuteLimit: minuteLimit, DefaultDailyLimit: dailyLimit}, nil
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

func TestApplicationAlreadySubmitted(t *testing.T) {
	store := &fakeApplicationStore{createErr: model.ErrApplicationExists}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != model.ErrApplicationExists {
		t.Fatalf("expected ErrApplicationExists, got %v", err)
	}
	if !store.created {
		t.Fatal("expected create path to be called")
	}
}

func TestCreateApplicationUsesDefaultLimits(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store)
	app, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test"})
	if err != nil {
		t.Fatalf("create application failed: %v", err)
	}
	if app.Status != model.ApplicationPending || app.DefaultMinuteLimit != 60 || app.DefaultDailyLimit != 5000 {
		t.Fatalf("unexpected created application: %+v", app)
	}
}

func TestListAdminNormalizesPageAndLimit(t *testing.T) {
	store := &fakeApplicationStore{}
	svc := NewService(config.Config{}, store)
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
	svc := NewService(config.Config{}, store)
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
