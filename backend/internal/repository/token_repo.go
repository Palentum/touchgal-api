package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type TokenRepo struct{ db Queryer }

func NewTokenRepo(db Queryer) *TokenRepo { return &TokenRepo{db: db} }

func (r *TokenRepo) Create(ctx context.Context, token model.APIToken) (*model.APIToken, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_tokens (id, user_id, application_id, name, token_prefix, token_hash, minute_limit, daily_limit, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, application_id, name, token_prefix, token_hash, status, minute_limit, daily_limit, last_used_at, expires_at, created_at, updated_at`,
		token.ID, token.UserID, token.ApplicationID, token.Name, token.TokenPrefix, token.TokenHash, token.MinuteLimit, token.DailyLimit, token.ExpiresAt,
	).Scan(&token.ID, &token.UserID, &token.ApplicationID, &token.Name, &token.TokenPrefix, &token.TokenHash, &token.Status, &token.MinuteLimit, &token.DailyLimit, &token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	return &token, err
}

func (r *TokenRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, application_id, name, token_prefix, token_hash, status, minute_limit, daily_limit, last_used_at, expires_at, created_at, updated_at
		FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTokens(rows)
}

func (r *TokenRepo) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.APIToken, error) {
	offset := (page - 1) * limit
	if status == "" {
		rows, err := r.db.Query(ctx, `
			SELECT id, user_id, application_id, name, token_prefix, token_hash, status, minute_limit, daily_limit, last_used_at, expires_at, created_at, updated_at
			FROM api_tokens ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanTokens(rows)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, application_id, name, token_prefix, token_hash, status, minute_limit, daily_limit, last_used_at, expires_at, created_at, updated_at
		FROM api_tokens WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTokens(rows)
}

func (r *TokenRepo) CountActive(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM api_tokens WHERE status = 'active'`).Scan(&count)
	return count, err
}

func (r *TokenRepo) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*model.APIToken, error) {
	return r.get(ctx, `WHERE id = $1 AND user_id = $2`, id, userID)
}

func (r *TokenRepo) GetByHashWithApplication(ctx context.Context, tokenHash string) (*model.TokenAuthInfo, error) {
	info := &model.TokenAuthInfo{}
	err := r.db.QueryRow(ctx, `
		SELECT t.id, t.user_id, t.application_id, t.name, t.token_prefix, t.token_hash, t.status, t.minute_limit, t.daily_limit, t.last_used_at, t.expires_at, t.created_at, t.updated_at,
		       a.status, a.id, a.user_id, u.minute_limit, u.daily_limit
		FROM api_tokens t
		JOIN api_applications a ON a.id = t.application_id
		JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1`, tokenHash,
	).Scan(&info.Token.ID, &info.Token.UserID, &info.Token.ApplicationID, &info.Token.Name, &info.Token.TokenPrefix, &info.Token.TokenHash, &info.Token.Status, &info.Token.MinuteLimit, &info.Token.DailyLimit, &info.Token.LastUsedAt, &info.Token.ExpiresAt, &info.Token.CreatedAt, &info.Token.UpdatedAt, &info.ApplicationStatus, &info.ApplicationID, &info.UserID, &info.UserMinuteLimit, &info.UserDailyLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return info, err
}

func (r *TokenRepo) UpdateNameForUser(ctx context.Context, id, userID uuid.UUID, name string) (*model.APIToken, error) {
	token := &model.APIToken{}
	err := r.db.QueryRow(ctx, `
		UPDATE api_tokens
		SET name = $3, updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, application_id, name, token_prefix, token_hash, status, minute_limit, daily_limit, last_used_at, expires_at, created_at, updated_at`,
		id, userID, name,
	).Scan(&token.ID, &token.UserID, &token.ApplicationID, &token.Name, &token.TokenPrefix, &token.TokenHash, &token.Status, &token.MinuteLimit, &token.DailyLimit, &token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return token, err
}

func (r *TokenRepo) DeleteForUser(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *TokenRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *TokenRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE api_tokens SET last_used_at = now() WHERE id = $1`, id)
	return err
}

func (r *TokenRepo) get(ctx context.Context, where string, args ...any) (*model.APIToken, error) {
	token := &model.APIToken{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, application_id, name, token_prefix, token_hash, status, minute_limit, daily_limit, last_used_at, expires_at, created_at, updated_at
		FROM api_tokens `+where, args...).Scan(&token.ID, &token.UserID, &token.ApplicationID, &token.Name, &token.TokenPrefix, &token.TokenHash, &token.Status, &token.MinuteLimit, &token.DailyLimit, &token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return token, err
}

func scanTokens(rows pgx.Rows) ([]model.APIToken, error) {
	tokens := []model.APIToken{}
	for rows.Next() {
		var token model.APIToken
		if err := rows.Scan(&token.ID, &token.UserID, &token.ApplicationID, &token.Name, &token.TokenPrefix, &token.TokenHash, &token.Status, &token.MinuteLimit, &token.DailyLimit, &token.LastUsedAt, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}
