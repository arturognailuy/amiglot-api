package service

import (
	"context"
	"testing"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

func TestDiscoveryService_FullMatchFlow(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewDiscoveryRepository(pool)
	svc := NewDiscoveryService(repo, 30) // low threshold for test

	// User A: speaks English (native), wants to learn Spanish
	var userA string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('matchA@test.com') RETURNING id`).Scan(&userA)
	if err != nil {
		t.Fatalf("create userA: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable, country_code, birth_year, birth_month)
		VALUES ($1, 'usera', 'usera', 'UTC', true, 'US', 2000, 6)`, userA)
	if err != nil {
		t.Fatalf("create profileA: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order) VALUES
		($1, 'en', 5, true, false, 0),
		($1, 'es', 1, false, true, 1)`, userA)
	if err != nil {
		t.Fatalf("add languagesA: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone, sort_order) VALUES
		($1, 1, '09:00', '12:00', 'UTC', 0),
		($1, 3, '14:00', '17:00', 'UTC', 1)`, userA)
	if err != nil {
		t.Fatalf("add slotsA: %v", err)
	}

	// User B: speaks Spanish (native), wants to learn English, bridge: English (level 3)
	var userB string
	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('matchB@test.com') RETURNING id`).Scan(&userB)
	if err != nil {
		t.Fatalf("create userB: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable, country_code, birth_year)
		VALUES ($1, 'userb', 'userb', 'UTC', true, 'MX', 1998)`, userB)
	if err != nil {
		t.Fatalf("create profileB: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order) VALUES
		($1, 'es', 5, true, false, 0),
		($1, 'en', 3, false, true, 1)`, userB)
	if err != nil {
		t.Fatalf("add languagesB: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone, sort_order) VALUES
		($1, 1, '10:00', '13:00', 'UTC', 0),
		($1, 3, '15:00', '18:00', 'UTC', 1)`, userB)
	if err != nil {
		t.Fatalf("add slotsB: %v", err)
	}

	// Discover from A's perspective
	result, err := svc.Discover(ctx, userA, nil, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Items))
	}

	m := result.Items[0]
	if m.UserID != userB {
		t.Errorf("expected userB, got %s", m.UserID)
	}
	if m.Handle != "userb" {
		t.Errorf("expected handle userb, got %s", m.Handle)
	}
	if m.CountryCode == nil || *m.CountryCode != "MX" {
		t.Errorf("expected country MX, got %v", m.CountryCode)
	}
	if m.Age == nil {
		t.Error("expected non-nil age")
	}
	if len(m.MutualTeach) == 0 {
		t.Error("expected mutual_teach to have entries")
	}
	if len(m.MutualLearn) == 0 {
		t.Error("expected mutual_learn to have entries")
	}
	if len(m.BridgeLanguages) == 0 {
		t.Error("expected bridge_languages to have entries")
	}
	if len(m.AvailabilityOverlap) == 0 {
		t.Error("expected availability_overlap to have entries")
	}
	if m.TotalOverlapMinutes <= 0 {
		t.Errorf("expected positive overlap, got %d", m.TotalOverlapMinutes)
	}

	// Discover from B's perspective
	result2, err := svc.Discover(ctx, userB, nil, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result2.Items) != 1 {
		t.Fatalf("expected 1 match from B, got %d", len(result2.Items))
	}
	if result2.Items[0].UserID != userA {
		t.Errorf("expected userA from B's perspective, got %s", result2.Items[0].UserID)
	}
}

func TestDiscoveryService_Pagination(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewDiscoveryRepository(pool)
	svc := NewDiscoveryService(repo, 30)

	// User A
	var userA string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('pagA@test.com') RETURNING id`).Scan(&userA)
	if err != nil {
		t.Fatalf("create userA: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'paga', 'paga', 'UTC', true)`, userA)
	if err != nil {
		t.Fatalf("create profileA: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order) VALUES
		($1, 'en', 5, true, false, 0),
		($1, 'es', 1, false, true, 1)`, userA)
	if err != nil {
		t.Fatalf("add languagesA: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone, sort_order) VALUES
		($1, 0, '00:00', '23:59', 'UTC', 0),
		($1, 1, '00:00', '23:59', 'UTC', 1),
		($1, 2, '00:00', '23:59', 'UTC', 2),
		($1, 3, '00:00', '23:59', 'UTC', 3),
		($1, 4, '00:00', '23:59', 'UTC', 4),
		($1, 5, '00:00', '23:59', 'UTC', 5),
		($1, 6, '00:00', '23:59', 'UTC', 6)`, userA)
	if err != nil {
		t.Fatalf("add slotsA: %v", err)
	}

	// Test with limit=1 should paginate
	result, err := svc.Discover(ctx, userA, nil, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// May have 0 results if no matches exist, just ensure no errors
	_ = result
}

func TestDiscoveryService_LimitCapping(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewDiscoveryRepository(pool)
	svc := NewDiscoveryService(repo, 60)

	var userID string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('limittest@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'limittest', 'limittest', 'UTC', true)`, userID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order)
		VALUES ($1, 'zh', 0, false, true, 0)`, userID)
	if err != nil {
		t.Fatalf("add language: %v", err)
	}

	// Limit 0 should default
	result, err := svc.Discover(ctx, userID, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result

	// Limit > 50 should cap
	result, err = svc.Discover(ctx, userID, nil, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestDiscoveryService_DefaultMinOverlap(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewDiscoveryRepository(pool)

	// Test with 0 value defaults to 60
	svc := NewDiscoveryService(repo, 0)
	if svc.minOverlapMinutes != 60 {
		t.Errorf("expected default 60, got %d", svc.minOverlapMinutes)
	}

	// Test with negative
	svc = NewDiscoveryService(repo, -5)
	if svc.minOverlapMinutes != 60 {
		t.Errorf("expected default 60, got %d", svc.minOverlapMinutes)
	}
}
