package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type adminApplicationRows struct {
	rows   []model.AdminApplication
	idx    int
	closed bool
	err    error
}

func (r *adminApplicationRows) Close()                                       { r.closed = true }
func (r *adminApplicationRows) Err() error                                   { return r.err }
func (r *adminApplicationRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *adminApplicationRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *adminApplicationRows) Next() bool {
	if r.idx >= len(r.rows) {
		r.closed = true
		return false
	}
	r.idx++
	return true
}
func (r *adminApplicationRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	*(dest[0].(*uuid.UUID)) = row.ID
	*(dest[1].(*uuid.UUID)) = row.UserID
	*(dest[2].(*string)) = row.ApplicantName
	*(dest[3].(*string)) = row.ProjectName
	*(dest[4].(*string)) = row.ProjectURL
	*(dest[5].(*int)) = row.ExpectedDailyRequests
	*(dest[6].(*string)) = row.UsageScenario
	*(dest[7].(*string)) = row.Status
	*(dest[8].(*int)) = row.DefaultMinuteLimit
	*(dest[9].(*int)) = row.DefaultDailyLimit
	*(dest[10].(**uuid.UUID)) = row.ReviewedBy
	*(dest[11].(**time.Time)) = row.ReviewedAt
	*(dest[12].(*time.Time)) = row.CreatedAt
	*(dest[13].(*time.Time)) = row.UpdatedAt
	*(dest[14].(*uuid.UUID)) = row.Owner.ID
	*(dest[15].(*string)) = row.Owner.Email
	*(dest[16].(*string)) = row.Owner.DisplayName
	return nil
}
func (r *adminApplicationRows) Values() ([]any, error) { return nil, nil }
func (r *adminApplicationRows) RawValues() [][]byte    { return nil }
func (r *adminApplicationRows) Conn() *pgx.Conn        { return nil }

func TestApplicationRepoListAdminIncludesOwnerAccount(t *testing.T) {
	applicationID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	rows := &adminApplicationRows{rows: []model.AdminApplication{{
		Application: model.Application{
			ID:                    applicationID,
			UserID:                userID,
			ApplicantName:         "Kun",
			ProjectName:           "Docs Bot",
			ProjectURL:            "https://example.com",
			ExpectedDailyRequests: 1000,
			UsageScenario:         "展示条目信息",
			Status:                model.ApplicationApproved,
			DefaultMinuteLimit:    60,
			DefaultDailyLimit:     5000,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
		Owner: model.ApplicationOwner{ID: userID, Email: "dev@example.com", DisplayName: "Dev"},
	}}}
	queryer := &queryOnlyQueryer{rows: rows}
	repo := NewApplicationRepo(queryer)

	apps, err := repo.ListAdmin(context.Background(), model.ApplicationApproved, 2, 10)
	if err != nil {
		t.Fatalf("list admin applications: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected one application, got %d", len(apps))
	}
	if apps[0].Owner.ID != userID || apps[0].Owner.Email != "dev@example.com" || apps[0].Owner.DisplayName != "Dev" {
		t.Fatalf("application owner account was not scanned: %#v", apps[0].Owner)
	}
	if !strings.Contains(queryer.sql, "JOIN users u ON u.id = a.user_id") || !strings.Contains(queryer.sql, "u.email::text") || !strings.Contains(queryer.sql, "u.display_name") {
		t.Fatalf("admin application query must include owner account columns: %q", queryer.sql)
	}
	if !strings.Contains(queryer.sql, "WHERE a.status = $1") {
		t.Fatalf("status filter must target application status: %q", queryer.sql)
	}
	if len(queryer.args) != 3 || queryer.args[0] != model.ApplicationApproved || queryer.args[1] != 10 || queryer.args[2] != 10 {
		t.Fatalf("unexpected admin application list args: %#v", queryer.args)
	}
	if !rows.closed {
		t.Fatal("admin application rows must be closed")
	}
}
