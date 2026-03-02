package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
)

func TestAuthServiceRequestMagicLink(t *testing.T) {
	pool := openTestPool(t)

	repo := repository.NewAuthRepository(pool)
	svc := NewAuthService(config.Config{Env: "dev", MagicLinkBaseURL: "http://localhost:3000/auth/verify", MagicLinkTTL: 10 * time.Minute}, repo)

	devURL, err := svc.RequestMagicLink(context.Background(), "")
	require.Error(t, err)
	var svcErr *Error
	require.ErrorAs(t, err, &svcErr)
	require.Equal(t, 400, svcErr.Status)
	require.Equal(t, "errors.email_required", svcErr.Key)
	require.Nil(t, devURL)

	devURL, err = svc.RequestMagicLink(context.Background(), "USER@EXAMPLE.COM")
	require.NoError(t, err)
	require.NotNil(t, devURL)
	require.Contains(t, *devURL, "token=")
}

func TestAuthServiceVerifyMagicLink(t *testing.T) {
	pool := openTestPool(t)

	repo := repository.NewAuthRepository(pool)
	svc := NewAuthService(config.Config{MagicLinkTTL: 10 * time.Minute}, repo)

	userID, err := repo.EnsureUser(context.Background(), "svc@example.com")
	require.NoError(t, err)

	token, tokenHash, err := GenerateToken()
	require.NoError(t, err)

	err = repo.CreateMagicLinkToken(context.Background(), userID, tokenHash, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	accessToken, gotUserID, email, err := svc.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.Equal(t, userID, gotUserID)
	require.Equal(t, "svc@example.com", email)
}

func TestAuthServiceNoPool(t *testing.T) {
	svc := NewAuthService(config.Config{}, repository.NewAuthRepository(nil))

	_, err := svc.RequestMagicLink(context.Background(), "user@example.com")
	require.Error(t, err)
	var svcErr *Error
	require.ErrorAs(t, err, &svcErr)
	require.Equal(t, 503, svcErr.Status)
}
