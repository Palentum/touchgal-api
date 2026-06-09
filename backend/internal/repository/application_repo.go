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

func (r *ApplicationRepo) Create(ctx context.Context, userID uuid.UUID, input model.CreateApplicationInput, minuteLimit, dailyLimit int) (*model.Application, error) {
	app := &model.Application{ID: uuid.New(), UserID: userID}
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_applications (id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, default_minute_limit, default_daily_limit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at`,
		app.ID, userID, input.ApplicantName, input.ProjectName, input.ProjectURL, input.ExpectedDailyRequests, input.UsageScenario, minuteLimit, dailyLimit,
	).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewNote, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, model.ErrApplicationExists
		}
	}
	return app, err
}

func (r *ApplicationRepo) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*model.Application, error) {
	return r.get(ctx, `WHERE id = $1 AND user_id = $2`, id, userID)
}

func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	return r.get(ctx, `WHERE id = $1`, id)
}

func (r *ApplicationRepo) GetApprovedByUser(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	return r.get(ctx, `WHERE user_id = $1 AND status = 'approved' ORDER BY reviewed_at DESC NULLS LAST, created_at DESC LIMIT 1`, userID)
}

func (r *ApplicationRepo) EnsureAdminApproved(ctx context.Context, userID uuid.UUID, minuteLimit, dailyLimit int) (*model.Application, error) {
	app := &model.Application{ID: uuid.New(), UserID: userID}
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_applications (id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_at)
		VALUES ($1, $2, 'Admin user', 'TouchGal API Admin', 'https://api.example.com/admin', $3, 'Admin account default approval', 'approved', $4, $5, 'Admin account default approval', now())
		ON CONFLICT (user_id) WHERE status <> 'rejected' DO UPDATE
		SET status = 'approved',
		    default_minute_limit = EXCLUDED.default_minute_limit,
		    default_daily_limit = EXCLUDED.default_daily_limit,
		    review_note = EXCLUDED.review_note,
		    reviewed_at = now()
		RETURNING id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at`,
		app.ID, userID, dailyLimit, minuteLimit, dailyLimit,
	).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewNote, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	return app, err
}

func (r *ApplicationRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows)
}

func (r *ApplicationRepo) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.Application, error) {
	offset := (page - 1) * limit
	if status == "" {
		rows, err := r.db.Query(ctx, `
			SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at
			FROM api_applications ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanApplications(rows)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows)
}

func (r *ApplicationRepo) UpdateReview(ctx context.Context, id, reviewer uuid.UUID, status, note string, minuteLimit, dailyLimit int) (*model.Application, error) {
	app := &model.Application{}
	err := r.db.QueryRow(ctx, `
		UPDATE api_applications
		SET status = $2, review_note = $3, reviewed_by = $4, reviewed_at = now(), default_minute_limit = $5, default_daily_limit = $6
		WHERE id = $1
		RETURNING id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at`,
		id, status, note, reviewer, minuteLimit, dailyLimit,
	).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewNote, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return app, err
}

func (r *ApplicationRepo) get(ctx context.Context, where string, args ...any) (*model.Application, error) {
	app := &model.Application{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, applicant_name, project_name, project_url, expected_daily_requests, usage_scenario, status, default_minute_limit, default_daily_limit, review_note, reviewed_by, reviewed_at, created_at, updated_at
		FROM api_applications `+where, args...).Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewNote, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return app, err
}

func scanApplications(rows pgx.Rows) ([]model.Application, error) {
	apps := []model.Application{}
	for rows.Next() {
		var app model.Application
		if err := rows.Scan(&app.ID, &app.UserID, &app.ApplicantName, &app.ProjectName, &app.ProjectURL, &app.ExpectedDailyRequests, &app.UsageScenario, &app.Status, &app.DefaultMinuteLimit, &app.DefaultDailyLimit, &app.ReviewNote, &app.ReviewedBy, &app.ReviewedAt, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}
