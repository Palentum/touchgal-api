package users

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

type fakeStore struct {
	listStatus string
	listQuery  string
	listPage   int
	listLimit  int

	updateID          uuid.UUID
	updateEmail       *string
	updateDisplayName *string
	updateStatus      *string
	updateMinuteLimit *int
	updateDailyLimit  *int
	updated           *model.User

	deletedID uuid.UUID
}

func (f *fakeStore) ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error) {
	f.listStatus = status
	f.listQuery = query
	f.listPage = page
	f.listLimit = limit
	return []model.User{{ID: uuid.New(), Status: model.UserStatusActive}}, nil
}

func (f *fakeStore) UpdateAdmin(ctx context.Context, id uuid.UUID, email, displayName, status *string, minuteLimit, dailyLimit *int) (*model.User, error) {
	f.updateID = id
	f.updateEmail = email
	f.updateDisplayName = displayName
	f.updateStatus = status
	f.updateMinuteLimit = minuteLimit
	f.updateDailyLimit = dailyLimit
	if f.updated != nil {
		return f.updated, nil
	}
	user := &model.User{ID: id}
	if email != nil {
		user.Email = *email
	}
	if displayName != nil {
		user.DisplayName = *displayName
	}
	if status != nil {
		user.Status = *status
	}
	if minuteLimit != nil {
		user.MinuteLimit = *minuteLimit
	}
	if dailyLimit != nil {
		user.DailyLimit = *dailyLimit
	}
	return user, nil
}

func (f *fakeStore) DeleteByID(ctx context.Context, id uuid.UUID) error {
	f.deletedID = id
	return nil
}

func TestListAdminNormalizesFiltersAndPage(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	users, err := svc.ListAdmin(context.Background(), " active ", "  dev@example.com  ", 0, 150)
	if err != nil {
		t.Fatalf("ListAdmin returned error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected one user, got %d", len(users))
	}
	if store.listStatus != model.UserStatusActive || store.listQuery != "dev@example.com" {
		t.Fatalf("unexpected filters: status=%q query=%q", store.listStatus, store.listQuery)
	}
	if store.listPage != 1 || store.listLimit != 100 {
		t.Fatalf("unexpected pagination: page=%d limit=%d", store.listPage, store.listLimit)
	}
}

func TestListAdminRejectsInvalidFilters(t *testing.T) {
	svc := NewService(&fakeStore{})
	if _, err := svc.ListAdmin(context.Background(), "pending", "", 1, 20); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid status error, got %v", err)
	}
	if _, err := svc.ListAdmin(context.Background(), "", strings.Repeat("a", maxSearchQueryLength+1), 1, 20); err != model.ErrInvalidInput {
		t.Fatalf("expected long query error, got %v", err)
	}
}

func TestUpdateAdminNormalizesStatus(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	actorID := uuid.New()
	targetID := uuid.New()
	status := " disabled "
	user, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{Status: &status})
	if err != nil {
		t.Fatalf("UpdateAdmin returned error: %v", err)
	}
	if user.ID != targetID {
		t.Fatalf("expected updated user id %s, got %s", targetID, user.ID)
	}
	if store.updateID != targetID {
		t.Fatalf("unexpected target id: %s", store.updateID)
	}
	if store.updateStatus == nil || *store.updateStatus != model.UserStatusDisabled {
		t.Fatalf("status was not normalized: %v", store.updateStatus)
	}
}

func TestUpdateAdminRejectsSelfDisable(t *testing.T) {
	svc := NewService(&fakeStore{})
	actorID := uuid.New()
	status := model.UserStatusDisabled
	if _, err := svc.UpdateAdmin(context.Background(), actorID, actorID, AdminUpdate{Status: &status}); err != model.ErrInvalidInput {
		t.Fatalf("expected self-disable rejection, got %v", err)
	}
}

func TestUpdateAdminRejectsInvalidStatus(t *testing.T) {
	svc := NewService(&fakeStore{})
	actorID := uuid.New()
	targetID := uuid.New()
	status := "pending"
	if _, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{Status: &status}); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid status rejection, got %v", err)
	}
}

func TestUpdateAdminNormalizesProfileAndLimits(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	actorID := uuid.New()
	targetID := uuid.New()
	email := " Dev@Example.COM "
	displayName := "  Developer  "
	status := " active "
	minuteLimit := 30
	dailyLimit := 3000
	user, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{
		Email:       &email,
		DisplayName: &displayName,
		Status:      &status,
		MinuteLimit: &minuteLimit,
		DailyLimit:  &dailyLimit,
	})
	if err != nil {
		t.Fatalf("UpdateAdmin returned error: %v", err)
	}
	if user.ID != targetID {
		t.Fatalf("expected updated user id %s, got %s", targetID, user.ID)
	}
	if store.updateEmail == nil || *store.updateEmail != "dev@example.com" {
		t.Fatalf("email was not normalized: %v", store.updateEmail)
	}
	if store.updateDisplayName == nil || *store.updateDisplayName != "Developer" {
		t.Fatalf("display name was not normalized: %v", store.updateDisplayName)
	}
	if store.updateStatus == nil || *store.updateStatus != model.UserStatusActive {
		t.Fatalf("status was not normalized: %v", store.updateStatus)
	}
	if store.updateMinuteLimit == nil || *store.updateMinuteLimit != minuteLimit || store.updateDailyLimit == nil || *store.updateDailyLimit != dailyLimit {
		t.Fatalf("limits were not passed through: minute=%v daily=%v", store.updateMinuteLimit, store.updateDailyLimit)
	}
}

func TestUpdateAdminRejectsInvalidProfileAndLimits(t *testing.T) {
	svc := NewService(&fakeStore{})
	actorID := uuid.New()
	targetID := uuid.New()
	if _, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{}); err != model.ErrInvalidInput {
		t.Fatalf("expected empty update rejection, got %v", err)
	}
	badEmail := "not-an-email"
	if _, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{Email: &badEmail}); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid email rejection, got %v", err)
	}
	longName := strings.Repeat("a", maxDisplayNameLength+1)
	if _, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{DisplayName: &longName}); err != model.ErrInvalidInput {
		t.Fatalf("expected long display name rejection, got %v", err)
	}
	minuteLimit := 100
	dailyLimit := 50
	if _, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{MinuteLimit: &minuteLimit, DailyLimit: &dailyLimit}); err != model.ErrInvalidInput {
		t.Fatalf("expected inverted limits rejection, got %v", err)
	}
	if _, err := svc.UpdateAdmin(context.Background(), actorID, targetID, AdminUpdate{MinuteLimit: &minuteLimit}); err != model.ErrInvalidInput {
		t.Fatalf("expected partial limits rejection, got %v", err)
	}
}

func TestDeleteAdminRejectsSelfDelete(t *testing.T) {
	svc := NewService(&fakeStore{})
	actorID := uuid.New()
	if err := svc.DeleteAdmin(context.Background(), actorID, actorID); err != model.ErrInvalidInput {
		t.Fatalf("expected self-delete rejection, got %v", err)
	}
}

func TestDeleteAdminDeletesTarget(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	actorID := uuid.New()
	targetID := uuid.New()
	if err := svc.DeleteAdmin(context.Background(), actorID, targetID); err != nil {
		t.Fatalf("DeleteAdmin returned error: %v", err)
	}
	if store.deletedID != targetID {
		t.Fatalf("unexpected deleted id: %s", store.deletedID)
	}
}
