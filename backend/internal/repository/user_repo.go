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
	user := &model.User{ID: uuid.New(), Email: email, DisplayName: displayName, Status: "active"}
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

func (r *UserRepo) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}
