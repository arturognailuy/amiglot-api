package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/config"
)

func TestNew_PingsDatabase(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatalf("DATABASE_URL must be set to run DB unit tests")
	}

	pool, err := New(config.Config{DatabaseURL: dbURL})
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Close()
}
