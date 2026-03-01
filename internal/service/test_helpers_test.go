package service

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testLockID int64 = 424242

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
		// Use a fresh context for cleanup just in case
		ctx := context.Background()
		_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, testLockID)
		conn.Release()
	})
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("..", "..", "db", "migrations", "*.sql"))
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
