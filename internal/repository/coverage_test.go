package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/model"
)

func TestPoolAccessors(t *testing.T) {
	pool := openTestPool(t)
	// We don't close pool here because openTestPool registers cleanup

	authRepo := NewAuthRepository(pool)
	require.NotNil(t, authRepo.Pool())
	require.Equal(t, pool, authRepo.Pool())

	profileRepo := NewProfileRepository(pool)
	require.NotNil(t, profileRepo.Pool())
	require.Equal(t, pool, profileRepo.Pool())
}

func TestIsUniqueViolation_Helpers(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"}
	require.True(t, IsUniqueViolation(err))

	err = &pgconn.PgError{Code: "23503"}
	require.False(t, IsUniqueViolation(err))

	require.False(t, IsUniqueViolation(errors.New("generic error")))
	require.False(t, IsUniqueViolation(nil))
}

func TestLoadLanguages_Coverage(t *testing.T) {
	pool := openTestPool(t)

	repo := NewProfileRepository(pool)
	var userID string
	err := pool.QueryRow(context.Background(), "INSERT INTO users (email) VALUES ('lang-test@example.com') RETURNING id").Scan(&userID)
	require.NoError(t, err)

	langs := []model.Language{
		{LanguageCode: "en", Level: 5, IsNative: true, IsTarget: false, Description: nil},
		{LanguageCode: "es", Level: 3, IsNative: false, IsTarget: true, Description: nil},
	}
	err = repo.ReplaceLanguages(context.Background(), userID, langs)
	require.NoError(t, err)

	loaded, err := repo.LoadLanguages(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	require.Equal(t, "en", loaded[0].LanguageCode)
	require.Equal(t, "es", loaded[1].LanguageCode)
}
