package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type ApplicationRepo struct{ db Queryer }

func NewApplicationRepo(db Queryer) *ApplicationRepo { return &ApplicationRepo{db: db} }

const applicationListByUserCapHint = 4

func (r *ApplicationRepo) Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput, minuteLimit, dailyLimit int) (*model.Application, error) {
	app := &model.Application{ID: uuid.New(), UserID: userID}
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_applications (id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, default_minute_limit, default_daily_limit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at`,
		app.ID, userID, input.ApplicantName, input.ProjectName, input.ProjectURL, input.ExpectedDailyRequests, input.UsageScenario, minuteLimit, dailyLimit,
	).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, model.ErrApplicationExists
		}
	}
	return app, err
}

func (r *ApplicationRepo) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*model.Application, error) {
	return scanApplicationRow(r.db.QueryRow(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications
		WHERE id = $1 AND user_id = $2`, id, userID))
}

func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	return scanApplicationRow(r.db.QueryRow(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications
		WHERE id = $1`, id))
}

func (r *ApplicationRepo) GetApprovedByUser(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	return scanApplicationRow(r.db.QueryRow(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications
		WHERE user_id = $1 AND status = 'approved'
		ORDER BY reviewed_at DESC NULLS LAST, created_at DESC
		LIMIT 1`, userID))
}

func (r *ApplicationRepo) EnsureAdminApproved(ctx context.Context, userID uuid.UUID, minuteLimit, dailyLimit int) (*model.Application, error) {
	app := &model.Application{ID: uuid.New(), UserID: userID}
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_applications (id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_at)
		VALUES ($1, $2, 'Admin user', 'TouchGal API Admin', 'https://api.example.com/admin', $3, 'Admin account default approval', 'approved', $4, $5, now())
		ON CONFLICT (user_id) WHERE status <> 'rejected' DO UPDATE
		SET status = 'approved',
		    default_minute_limit = EXCLUDED.default_minute_limit,
		    default_daily_limit = EXCLUDED.default_daily_limit,
		    reviewed_at = now()
		RETURNING id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at`,
		app.ID, userID, dailyLimit, minuteLimit, dailyLimit,
	).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	return app, err
}

func (r *ApplicationRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows, applicationListByUserCapHint)
}

func (r *ApplicationRepo) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminApplication, error) {
	offset := (page - 1) * limit
	if status == "" {
		rows, err := r.db.Query(ctx, `
			SELECT a.id, a.user_id, a.applicant_name, a.project_name, a.project_url, a.expected_daily_requests, a.usage_scenario, a.status, a.default_minute_limit, a.default_daily_limit, a.reviewed_by, a.reviewed_at, a.created_at, a.updated_at,
			       u.id, u.email::text, u.display_name
			FROM api_applications a
			JOIN users u ON u.id = a.user_id
			ORDER BY a.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanAdminApplications(rows, limit)
	}
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.user_id, a.applicant_name, a.project_name, a.project_url, a.expected_daily_requests, a.usage_scenario, a.status, a.default_minute_limit, a.default_daily_limit, a.reviewed_by, a.reviewed_at, a.created_at, a.updated_at,
		       u.id, u.email::text, u.display_name
		FROM api_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.status = $1 ORDER BY a.created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdminApplications(rows, limit)
}

func (r *ApplicationRepo) UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status string, minuteLimit, dailyLimit int) (*model.Application, error) {
	app := &model.Application{}
	err := r.db.QueryRow(ctx, `
		UPDATE api_applications
		SET status = $2, reviewed_by = $3, reviewed_at = now(), default_minute_limit = $4, default_daily_limit = $5
		WHERE id = $1
		RETURNING id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, reviewed_by, reviewed_at, created_at, updated_at`,
		id, status, reviewer, minuteLimit, dailyLimit,
	).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return app, err
}

func scanApplicationRow(row pgx.Row) (*model.Application, error) {
	app := &model.Application{}
	err := row.Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return app, err
}

func scanApplications(rows pgx.Rows, capHint int) ([]model.Application, error) {
	apps := make([]model.Application, 0)
	for rows.Next() {
		var app model.Application
		if err := rows.Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		if cap(apps) == 0 {
			apps = make([]model.Application, 0, positiveCapHint(capHint))
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func scanAdminApplications(rows pgx.Rows, capHint int) ([]model.AdminApplication, error) {
	apps := make([]model.AdminApplication, 0)
	for rows.Next() {
		var app model.AdminApplication
		if err := rows.Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt, &app.Owner.ID, &app.Owner.Email, &app.Owner.DisplayName); err != nil {
			return nil, err
		}
		if cap(apps) == 0 {
			apps = make([]model.AdminApplication, 0, positiveCapHint(capHint))
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}
