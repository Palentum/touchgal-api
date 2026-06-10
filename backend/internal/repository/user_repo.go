package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type UserRepo struct{ db Queryer }

func NewUserRepo(db Queryer) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(ctx context.Context, email, displayName string) (*model.User, error) {
	user := &model.User{ID: uuid.New(), Email: email, DisplayName: displayName, Status: model.UserStatusActive}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (id, email, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, display_name, status, is_admin, minute_limit, daily_limit, last_login_at, created_at, updated_at`,
		user.ID, email, displayName,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.MinuteLimit, &user.DailyLimit, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, display_name, status, is_admin, minute_limit, daily_limit, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.MinuteLimit, &user.DailyLimit, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return user, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, display_name, status, is_admin, minute_limit, daily_limit, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.MinuteLimit, &user.DailyLimit, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return user, err
}

func (r *UserRepo) ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx, `
		SELECT id, email, display_name, status, is_admin, minute_limit, daily_limit, last_login_at, created_at, updated_at
		FROM users
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR email ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%')
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, status, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows, limit)
}

func (r *UserRepo) UpdateAdmin(ctx context.Context, id uuid.UUID, email, displayName, status *string, minuteLimit, dailyLimit *int) (*model.User, error) {
	user := &model.User{}
	emailArg := stringArg(email)
	displayNameArg := stringArg(displayName)
	statusArg := stringArg(status)
	minuteLimitArg := intArg(minuteLimit)
	dailyLimitArg := intArg(dailyLimit)
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET email = COALESCE($2, email),
		    display_name = COALESCE($3, display_name),
		    status = COALESCE($4, status),
		    minute_limit = COALESCE($5, minute_limit),
		    daily_limit = COALESCE($6, daily_limit)
		WHERE id = $1
		RETURNING id, email, display_name, status, is_admin, minute_limit, daily_limit, last_login_at, created_at, updated_at`,
		id, emailArg, displayNameArg, statusArg, minuteLimitArg, dailyLimitArg,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.MinuteLimit, &user.DailyLimit, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, model.ErrConflict
	}
	return user, err
}

func stringArg(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func intArg(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func (r *UserRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *UserRepo) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

func scanUsers(rows pgx.Rows, capHint int) ([]model.User, error) {
	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.MinuteLimit, &user.DailyLimit, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		if cap(users) == 0 {
			users = make([]model.User, 0, positiveCapHint(capHint))
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
