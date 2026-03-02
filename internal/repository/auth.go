package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *AuthRepository) EnsureUser(ctx context.Context, email string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	err = r.pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *AuthRepository) CreateMagicLinkToken(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO magic_link_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID,
		tokenHash,
		expiresAt,
	)
	return err
}

func (r *AuthRepository) ConsumeMagicLinkToken(ctx context.Context, tokenHash []byte) (string, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var tokenID string
	var userID string
	row := tx.QueryRow(ctx,
		`SELECT id, user_id FROM magic_link_tokens
		 WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
		 FOR UPDATE`,
		tokenHash,
	)
	if err := row.Scan(&tokenID, &userID); err != nil {
		return "", "", err
	}

	if _, err := tx.Exec(ctx, `UPDATE magic_link_tokens SET consumed_at = now() WHERE id = $1`, tokenID); err != nil {
		return "", "", err
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID); err != nil {
		return "", "", err
	}

	var email string
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		return "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}

	return userID, email, nil
}
