package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeApplicationStore struct {
	count   int
	created bool
}

func (f *fakeApplicationStore) Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput, minuteLimit, dailyLimit int) (*model.Application, error) {
	f.created = true
	return &model.Application{ID: uuid.New(), UserID: userID, Status: model.ApplicationPending, DefaultMinuteLimit: minuteLimit, DefaultDailyLimit: dailyLimit}, nil
}
func (f *fakeApplicationStore) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return f.count, nil
}
func (f *fakeApplicationStore) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	return nil, nil
}
func (f *fakeApplicationStore) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.Application, error) {
	return nil, nil
}
func (f *fakeApplicationStore) UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status, note string, minuteLimit, dailyLimit int) (*model.Application, error) {
	return nil, nil
}

func TestApplicationSubmittedOnce(t *testing.T) {
	store := &fakeApplicationStore{count: 1}
	svc := NewService(config.Config{DefaultTokenMinuteLimit: 60, DefaultTokenDailyLimit: 5000}, store)
	_, err := svc.Create(context.Background(), uuid.New(), model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 1, UsageScenario: "test", AgreeToTerms: true})
	if err != model.ErrApplicationExists {
		t.Fatalf("expected ErrApplicationExists, got %v", err)
	}
	if store.created {
		t.Fatal("must not create second account application")
	}
}

func TestValidateApplicationInput(t *testing.T) {
	input := model.CreateApplicationInput{ApplicantName: "Kun", ProjectURL: "https://example.com", ExpectedDailyRequests: 100, UsageScenario: "用于展示条目信息", AgreeToTerms: true}
	if err := ValidateInput(&input); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	bad := input
	bad.ProjectURL = "javascript:alert(1)"
	if err := ValidateInput(&bad); err == nil {
		t.Fatal("invalid URL accepted")
	}
}
