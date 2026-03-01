package repository

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuthRepositoryFlow(t *testing.T) {
	pool := openTestPool(t)

	repo := NewAuthRepository(pool)

	userID, err := repo.EnsureUser(context.Background(), "repo-user@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, userID)

	sameUserID, err := repo.EnsureUser(context.Background(), "repo-user@example.com")
	require.NoError(t, err)
	require.Equal(t, userID, sameUserID)

	token := "token-value"
	tokenHash := sha256.Sum256([]byte(token))
	expiresAt := time.Now().Add(1 * time.Hour)
	err = repo.CreateMagicLinkToken(context.Background(), userID, tokenHash[:], expiresAt)
	require.NoError(t, err)

	consumedUserID, email, err := repo.ConsumeMagicLinkToken(context.Background(), tokenHash[:])
	require.NoError(t, err)
	require.Equal(t, userID, consumedUserID)
	require.Equal(t, "repo-user@example.com", email)
}
