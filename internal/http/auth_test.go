package http

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

func newAuthHandler(pool *pgxpool.Pool, cfg config.Config) *authHandler {
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(cfg, repo)
	return &authHandler{svc: svc}
}

func TestRequestMagicLink_Validation(t *testing.T) {
	pool := openTestPool(t)

	h := newAuthHandler(pool, config.Config{MagicLinkTTL: 15 * time.Minute})

	_, err := h.requestMagicLink(context.Background(), &magicLinkRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "email is required")
}

func TestRequestMagicLink_DevFlow(t *testing.T) {
	pool := openTestPool(t)

	h := newAuthHandler(pool, config.Config{Env: "dev", MagicLinkBaseURL: "http://localhost:3000/auth/verify", MagicLinkTTL: 10 * time.Minute})

	resp, err := h.requestMagicLink(context.Background(), &magicLinkRequest{Body: struct {
		Email string `json:"email"`
	}{Email: "USER@EXAMPLE.COM"}})
	require.NoError(t, err)
	require.True(t, resp.Body.Ok)
	require.NotNil(t, resp.Body.DevLoginURL)
	require.Contains(t, *resp.Body.DevLoginURL, "token=")

	var count int
	err = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM magic_link_tokens`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestVerifyMagicLink_Flows(t *testing.T) {
	pool := openTestPool(t)

	repo := repository.NewAuthRepository(pool)
	h := newAuthHandler(pool, config.Config{MagicLinkTTL: 10 * time.Minute})

	userID, err := repo.EnsureUser(context.Background(), "user@example.com")
	require.NoError(t, err)

	token, tokenHash, err := service.GenerateToken()
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `
    INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
    VALUES ($1, $2, now() + interval '1 hour')
  `, userID, tokenHash)
	require.NoError(t, err)

	resp, err := h.verifyMagicLink(context.Background(), &verifyRequest{Body: struct {
		Token string `json:"token"`
	}{Token: token}})
	require.NoError(t, err)
	require.Equal(t, userID, resp.Body.User.ID)
	require.Equal(t, "user@example.com", resp.Body.User.Email)
	require.NotEmpty(t, resp.Body.AccessToken)

	var consumedAt *time.Time
	err = pool.QueryRow(context.Background(), `SELECT consumed_at FROM magic_link_tokens WHERE token_hash = $1`, tokenHash).Scan(&consumedAt)
	require.NoError(t, err)
	require.NotNil(t, consumedAt)

	_, err = h.verifyMagicLink(context.Background(), &verifyRequest{Body: struct {
		Token string `json:"token"`
	}{Token: "bad-token"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid or expired token")
}

func TestRequestMagicLink_NoPool(t *testing.T) {
	h := newAuthHandler(nil, config.Config{MagicLinkTTL: 5 * time.Minute})

	_, err := h.requestMagicLink(context.Background(), &magicLinkRequest{Body: struct {
		Email string `json:"email"`
	}{Email: "user@example.com"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "database unavailable")
}

func TestVerifyMagicLink_NoPool(t *testing.T) {
	h := newAuthHandler(nil, config.Config{MagicLinkTTL: 5 * time.Minute})

	_, err := h.verifyMagicLink(context.Background(), &verifyRequest{Body: struct {
		Token string `json:"token"`
	}{Token: "token"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "database unavailable")
}

func TestGenerateToken(t *testing.T) {
	first, firstHash, err := service.GenerateToken()
	require.NoError(t, err)
	require.NotEmpty(t, first)
	require.Len(t, firstHash, 32)

	second, secondHash, err := service.GenerateToken()
	require.NoError(t, err)
	require.NotEmpty(t, second)
	require.Len(t, secondHash, 32)
	require.NotEqual(t, first, second)
}
