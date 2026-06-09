package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type AuthRepo struct{ db Queryer }

func NewAuthRepo(db Queryer) *AuthRepo { return &AuthRepo{db: db} }

func (r *AuthRepo) InsertCode(ctx context.Context, email, purpose, codeHash, ip string, expiresAt time.Time) (*model.EmailVerificationCode, error) {
	code := &model.EmailVerificationCode{ID: uuid.New(), Email: email, Purpose: purpose, CodeHash: codeHash, IP: ip, ExpiresAt: expiresAt}
	err := r.db.QueryRow(ctx, `
		INSERT INTO email_verification_codes (id, email, purpose, code_hash, expires_at, ip)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, purpose, code_hash, expires_at, consumed_at, attempts, ip, created_at`,
		code.ID, email, purpose, codeHash, expiresAt, ip,
	).Scan(&code.ID, &code.Email, &code.Purpose, &code.CodeHash, &code.ExpiresAt, &code.ConsumedAt, &code.Attempts, &code.IP, &code.CreatedAt)
	return code, err
}

func (r *AuthRepo) LatestCode(ctx context.Context, email, purpose string) (*model.EmailVerificationCode, error) {
	code := &model.EmailVerificationCode{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, purpose, code_hash, expires_at, consumed_at, attempts, ip, created_at
		FROM email_verification_codes
		WHERE email = $1 AND purpose = $2
		ORDER BY created_at DESC
		LIMIT 1`, email, purpose,
	).Scan(&code.ID, &code.Email, &code.Purpose, &code.CodeHash, &code.ExpiresAt, &code.ConsumedAt, &code.Attempts, &code.IP, &code.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	return code, err
}

func (r *AuthRepo) IncrementCodeAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verification_codes SET attempts = attempts + 1 WHERE id = $1`, id)
	return err
}

func (r *AuthRepo) ConsumeCode(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verification_codes SET consumed_at = now() WHERE id = $1 AND consumed_at IS NULL`, id)
	return err
}

func (r *AuthRepo) CreateSession(ctx context.Context, userID uuid.UUID, sessionHash, userAgent, ip string, expiresAt time.Time) (*model.Session, error) {
	session := &model.Session{ID: uuid.New(), UserID: userID, SessionHash: sessionHash, UserAgent: userAgent, IP: ip, ExpiresAt: expiresAt}
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (id, user_id, session_hash, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, session_hash, user_agent, ip, expires_at, revoked_at, created_at, last_seen_at`,
		session.ID, userID, sessionHash, userAgent, ip, expiresAt,
	).Scan(&session.ID, &session.UserID, &session.SessionHash, &session.UserAgent, &session.IP, &session.ExpiresAt, &session.RevokedAt, &session.CreatedAt, &session.LastSeenAt)
	return session, err
}

func (r *AuthRepo) GetSessionUser(ctx context.Context, sessionHash string, now time.Time) (*model.Session, *model.User, error) {
	session := &model.Session{}
	user := &model.User{}
	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.session_hash, s.user_agent, s.ip, s.expires_at, s.revoked_at, s.created_at, s.last_seen_at,
		       u.id, u.email, u.display_name, u.status, u.is_admin, u.minute_limit, u.daily_limit, u.last_login_at, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_hash = $1 AND s.expires_at > $2 AND s.revoked_at IS NULL`,
		sessionHash, now,
	).Scan(&session.ID, &session.UserID, &session.SessionHash, &session.UserAgent, &session.IP, &session.ExpiresAt, &session.RevokedAt, &session.CreatedAt, &session.LastSeenAt, &user.ID, &user.Email, &user.DisplayName, &user.Status, &user.IsAdmin, &user.MinuteLimit, &user.DailyLimit, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, model.ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	_, _ = r.db.Exec(ctx, `UPDATE sessions SET last_seen_at = now() WHERE id = $1`, session.ID)
	return session, user, nil
}

func (r *AuthRepo) RevokeSession(ctx context.Context, sessionHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE session_hash = $1 AND revoked_at IS NULL`, sessionHash)
	return err
}
