package sqlc

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const testLockID int64 = 424242

func TestQueries_UserAndTokenFlow(t *testing.T) {
	pool := openTestPool(t)

	q := New(pool)

	user, err := q.CreateUser(context.Background(), "sqlc@example.com")
	require.NoError(t, err)

	fetched, err := q.GetUserByEmail(context.Background(), "sqlc@example.com")
	require.NoError(t, err)
	require.Equal(t, user.ID, fetched.ID)

	token, err := q.CreateMagicLinkToken(context.Background(), CreateMagicLinkTokenParams{
		UserID:    user.ID,
		TokenHash: []byte("hash"),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	})
	require.NoError(t, err)
	require.False(t, token.ConsumedAt.Valid)

	valid, err := q.GetValidMagicLinkToken(context.Background(), []byte("hash"))
	require.NoError(t, err)
	require.Equal(t, token.ID, valid.ID)

	err = q.ConsumeMagicLinkToken(context.Background(), token.ID)
	require.NoError(t, err)
}

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatalf("DATABASE_URL must be set to run DB unit tests")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	acquireTestLock(t, pool)
	applyMigrations(t, pool)
	resetTables(t, pool)

	return pool
}

func acquireTestLock(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock conn: %v", err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, testLockID); err != nil {
		conn.Release()
		t.Fatalf("acquire db lock: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, testLockID)
		conn.Release()
	})
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "db", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	sort.Strings(paths)

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}

		parts := strings.Split(string(content), "-- +goose Down")
		up := parts[0]
		up = strings.ReplaceAll(up, "-- +goose Up", "")
		if strings.TrimSpace(up) == "" {
			continue
		}

		if _, err := pool.Exec(context.Background(), up); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
}

func resetTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
    TRUNCATE TABLE
      magic_link_tokens,
      availability_slots,
      user_languages,
      profiles,
      users
    RESTART IDENTITY CASCADE;
  `)
	if err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}

func TestQueries_GetUserByID_UpdateUserLastLogin(t *testing.T) {
	pool := openTestPool(t)

	q := New(pool)
	user, err := q.CreateUser(context.Background(), "update@example.com")
	require.NoError(t, err)

	fetched, err := q.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, fetched.Email)

	err = q.UpdateUserLastLogin(context.Background(), user.ID)
	require.NoError(t, err)

	var lastLogin *time.Time
	err = pool.QueryRow(context.Background(), `SELECT last_login_at FROM users WHERE id = $1`, user.ID).Scan(&lastLogin)
	require.NoError(t, err)
	require.NotNil(t, lastLogin)
}

func TestQueries_WithTx(t *testing.T) {
	pool := openTestPool(t)

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := New(pool).WithTx(tx)
	user, err := qtx.CreateUser(ctx, "tx@example.com")
	require.NoError(t, err)

	fetched, err := qtx.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, fetched.Email)
}
