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

	updateID     uuid.UUID
	updateStatus string
	updated      *model.User
}

func (f *fakeStore) ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error) {
	f.listStatus = status
	f.listQuery = query
	f.listPage = page
	f.listLimit = limit
	return []model.User{{ID: uuid.New(), Status: model.UserStatusActive}}, nil
}

func (f *fakeStore) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.User, error) {
	f.updateID = id
	f.updateStatus = status
	if f.updated != nil {
		return f.updated, nil
	}
	return &model.User{ID: id, Status: status}, nil
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

func TestUpdateStatusNormalizesStatus(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	actorID := uuid.New()
	targetID := uuid.New()
	user, err := svc.UpdateStatus(context.Background(), actorID, targetID, " disabled ")
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if user.ID != targetID {
		t.Fatalf("expected updated user id %s, got %s", targetID, user.ID)
	}
	if store.updateID != targetID {
		t.Fatalf("unexpected target id: %s", store.updateID)
	}
	if store.updateStatus != model.UserStatusDisabled {
		t.Fatalf("status was not normalized: %q", store.updateStatus)
	}
}

func TestUpdateStatusRejectsSelfDisable(t *testing.T) {
	svc := NewService(&fakeStore{})
	actorID := uuid.New()
	if _, err := svc.UpdateStatus(context.Background(), actorID, actorID, model.UserStatusDisabled); err != model.ErrInvalidInput {
		t.Fatalf("expected self-disable rejection, got %v", err)
	}
}

func TestUpdateStatusRejectsInvalidStatus(t *testing.T) {
	svc := NewService(&fakeStore{})
	actorID := uuid.New()
	targetID := uuid.New()
	if _, err := svc.UpdateStatus(context.Background(), actorID, targetID, "pending"); err != model.ErrInvalidInput {
		t.Fatalf("expected invalid status rejection, got %v", err)
	}
}
