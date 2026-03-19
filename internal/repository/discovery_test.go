package repository

import (
	"context"
	"testing"
)

func TestDiscoveryRepository_DiscoverMatches_Empty(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := NewDiscoveryRepository(pool)

	// Create a user
	var userID string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('discover-repo@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	matches, err := repo.DiscoverMatches(ctx, userID, 60, 20, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestDiscoveryRepository_IsDiscoverable(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := NewDiscoveryRepository(pool)

	// Non-existent user
	ok, err := repo.IsDiscoverable(ctx, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected not discoverable for non-existent user")
	}

	// Create user + profile
	var userID string
	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('disc-repo-test@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'discrepo', 'discrepo', 'UTC', true)`, userID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	ok, err = repo.IsDiscoverable(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected discoverable")
	}
}

func TestDiscoveryRepository_HasTargetLanguages(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := NewDiscoveryRepository(pool)

	var userID string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('targets-repo@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	has, err := repo.HasTargetLanguages(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("expected no target languages")
	}

	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order)
		VALUES ($1, 'ja', 1, false, true, 0)`, userID)
	if err != nil {
		t.Fatalf("add language: %v", err)
	}

	has, err = repo.HasTargetLanguages(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("expected target languages")
	}
}

func TestDiscoveryRepository_GetUserLanguages(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := NewDiscoveryRepository(pool)

	var userID string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('langs-repo@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	langs, err := repo.GetUserLanguages(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(langs) != 0 {
		t.Errorf("expected 0 langs, got %d", len(langs))
	}

	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order)
		VALUES ($1, 'en', 5, true, false, 0), ($1, 'es', 2, false, true, 1)`, userID)
	if err != nil {
		t.Fatalf("add languages: %v", err)
	}

	langs, err = repo.GetUserLanguages(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(langs) != 2 {
		t.Errorf("expected 2 langs, got %d", len(langs))
	}
	if langs[0].LanguageCode != "en" || langs[0].Level != 5 || !langs[0].IsNative {
		t.Errorf("unexpected first lang: %+v", langs[0])
	}
	if langs[1].LanguageCode != "es" || !langs[1].IsTarget {
		t.Errorf("unexpected second lang: %+v", langs[1])
	}
}

func TestDiscoveryRepository_GetOverlapDetails_NoOverlap(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := NewDiscoveryRepository(pool)

	var u1, u2 string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('overlap1@test.com') RETURNING id`).Scan(&u1)
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('overlap2@test.com') RETURNING id`).Scan(&u2)
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	overlaps, err := repo.GetOverlapDetails(ctx, u1, u2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(overlaps) != 0 {
		t.Errorf("expected 0 overlaps, got %d", len(overlaps))
	}
}

func TestDiscoveryRepository_Pool(t *testing.T) {
	pool := openTestPool(t)
	repo := NewDiscoveryRepository(pool)
	if repo.Pool() != pool {
		t.Error("Pool() should return the same pool")
	}
}
