package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type UserRepo struct{ db Queryer }

func NewUserRepo(db Queryer) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(ctx context.Context, email, displayName string) (*model.User, error) {
	user := &model.User{ID: uuid.New(), Email: email, DisplayName: displayName, Status: model.UserStatusActive}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (id, email, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, display_name, status, is_admin, last_login_at, created_at, updated_at`,
		user.ID, email, displayName,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, display_name, status, is_admin, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return user, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, display_name, status, is_admin, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return user, err
}

func (r *UserRepo) ListAdmin(ctx context.Context, status, query string, page, limit int) ([]model.User, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx, `
		SELECT id, email, display_name, status, is_admin, last_login_at, created_at, updated_at
		FROM users
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR email ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%')
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, status, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *UserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET status = $2
		WHERE id = $1
		RETURNING id, email, display_name, status, is_admin, last_login_at, created_at, updated_at`,
		id, status,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return user, err
}

func (r *UserRepo) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

func scanUsers(rows pgx.Rows) ([]model.User, error) {
	users := []model.User{}
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
