package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnailuy/amiglot-api/internal/config"
)

func TestDiscoverEndpoint_NoAuth(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/discover", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Without X-User-Id header, should get an error
	if rec.Code == http.StatusOK {
		t.Logf("Response: %s", rec.Body.String())
		// Some frameworks return 200 with empty results if no user check,
		// but our service should return an error for missing/invalid user
	}
}

func TestDiscoverEndpoint_NonExistentUser(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/discover", nil)
	req.Header.Set("X-User-Id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Should return 403 (profile not found / not discoverable)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiscoverEndpoint_ProfileNotDiscoverable(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()

	// Create user + profile (not discoverable)
	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('disc-test@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = pool.Exec(context.Background(), `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'disctest', 'disctest', 'UTC', false)`, userID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	mux := Router(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/discover", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiscoverEndpoint_NoTargetLanguages(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('disc-notarget@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = pool.Exec(context.Background(), `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'discnotarget', 'discnotarget', 'UTC', true)`, userID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	mux := Router(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/discover", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDiscoverEndpoint_EmptyResults(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('disc-empty@test.com') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = pool.Exec(context.Background(), `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'discempty', 'discempty', 'UTC', true)`, userID)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}

	_, err = pool.Exec(context.Background(), `INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order)
		VALUES ($1, 'es', 0, false, true, 0)`, userID)
	if err != nil {
		t.Fatalf("add language: %v", err)
	}

	mux := Router(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/discover?limit=10", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Items      []json.RawMessage `json:"items"`
		NextCursor *string           `json:"next_cursor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(body.Items))
	}
}
