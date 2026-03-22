package service

import (
	"context"
	"testing"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

func TestDiscoveryService_Errors(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewDiscoveryRepository(pool)
	svc := NewDiscoveryService(repo, 60)

	// Create a user without a profile (not discoverable)
	var userID string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('discover@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Should fail: no profile → ERR_PROFILE_INCOMPLETE
	_, err = svc.Discover(ctx, userID, nil, 20)
	if err == nil {
		t.Fatal("expected error for non-discoverable user")
	}
	svcErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected service.Error, got %T", err)
	}
	if svcErr.Status != 403 || svcErr.Key != "errors.profile_incomplete" {
		t.Errorf("expected 403/profile_incomplete, got %d/%s", svcErr.Status, svcErr.Key)
	}

	// Create profile but not discoverable
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'tester1', 'tester1', 'UTC', false)`, userID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	_, err = svc.Discover(ctx, userID, nil, 20)
	svcErr, ok = err.(*Error)
	if !ok || svcErr.Status != 403 {
		t.Errorf("expected 403, got %v", err)
	}

	// Make discoverable but no target languages
	_, err = pool.Exec(ctx, `UPDATE profiles SET discoverable = true WHERE user_id = $1`, userID)
	if err != nil {
		t.Fatalf("update discoverable: %v", err)
	}

	_, err = svc.Discover(ctx, userID, nil, 20)
	svcErr, ok = err.(*Error)
	if !ok || svcErr.Status != 422 || svcErr.Key != "errors.no_target_languages" {
		t.Errorf("expected 422/no_target_languages, got %v", err)
	}

	// Add a target language → should succeed with empty results
	_, err = pool.Exec(ctx, `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order)
		VALUES ($1, 'es', 0, false, true, 0)`, userID)
	if err != nil {
		t.Fatalf("add language: %v", err)
	}

	result, err := svc.Discover(ctx, userID, nil, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestDiscoveryService_NilRepo(t *testing.T) {
	svc := NewDiscoveryService(nil, 60)
	_, err := svc.Discover(context.Background(), "some-id", nil, 20)
	if err == nil {
		t.Fatal("expected error for nil repo")
	}
}

func TestComputeAge(t *testing.T) {
	year := 2000
	month := int16(1)

	age := computeAge(&year, &month)
	if age == nil || *age < 25 {
		t.Errorf("expected age >= 25, got %v", age)
	}

	age = computeAge(nil, nil)
	if age != nil {
		t.Errorf("expected nil age, got %v", age)
	}
}
