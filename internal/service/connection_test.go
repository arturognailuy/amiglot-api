package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

func TestConnectionService_CreateMatchRequest(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewConnectionRepository(pool)
	svc := NewConnectionService(repo, 5, 500)

	// Create two discoverable users
	userA := createConnTestUser(t, ctx, pool, "conn-a@test.com", "conntesta")
	userB := createConnTestUser(t, ctx, pool, "conn-b@test.com", "conntestb")

	// Basic create
	msg := "Hello!"
	result, err := svc.CreateMatchRequest(ctx, userA, userB, &msg)
	if err != nil {
		t.Fatalf("create match request: %v", err)
	}
	if result.Status != "pending" {
		t.Errorf("expected pending, got %s", result.Status)
	}
	if result.RequesterID != userA || result.RecipientID != userB {
		t.Errorf("wrong IDs: requester=%s recipient=%s", result.RequesterID, result.RecipientID)
	}

	// Self request
	_, err = svc.CreateMatchRequest(ctx, userA, userA, nil)
	assertServiceError(t, err, 400, "errors.self_request")

	// Duplicate pending request
	_, err = svc.CreateMatchRequest(ctx, userA, userB, nil)
	assertServiceError(t, err, 409, "errors.request_exists")

	// Non-existent recipient
	_, err = svc.CreateMatchRequest(ctx, userA, "00000000-0000-0000-0000-000000000000", nil)
	assertServiceError(t, err, 404, "errors.user_not_found")

	// Message too long
	longMsg := make([]byte, 501)
	for i := range longMsg {
		longMsg[i] = 'a'
	}
	longStr := string(longMsg)
	_, err = svc.CreateMatchRequest(ctx, userB, userA, &longStr)
	assertServiceError(t, err, 400, "errors.message_too_long")
}

func TestConnectionService_AcceptDeclineCancel(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewConnectionRepository(pool)
	svc := NewConnectionService(repo, 5, 500)

	userA := createConnTestUser(t, ctx, pool, "accept-a@test.com", "accepta")
	userB := createConnTestUser(t, ctx, pool, "accept-b@test.com", "acceptb")
	userC := createConnTestUser(t, ctx, pool, "accept-c@test.com", "acceptc")

	// Accept flow
	msg := "Hi!"
	req, err := svc.CreateMatchRequest(ctx, userA, userB, &msg)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Requester can't accept
	_, err = svc.AcceptMatchRequest(ctx, req.ID, userA)
	assertServiceError(t, err, 403, "errors.not_recipient")

	// Third party can't accept
	_, err = svc.AcceptMatchRequest(ctx, req.ID, userC)
	assertServiceError(t, err, 403, "errors.not_recipient")

	// Recipient accepts
	matchID, err := svc.AcceptMatchRequest(ctx, req.ID, userB)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if matchID == "" {
		t.Error("expected match ID")
	}

	// Can't accept again
	_, err = svc.AcceptMatchRequest(ctx, req.ID, userB)
	assertServiceError(t, err, 409, "errors.not_pending")

	// Already matched
	_, err = svc.CreateMatchRequest(ctx, userA, userB, nil)
	assertServiceError(t, err, 409, "errors.already_matched")

	// Decline flow
	req2, err := svc.CreateMatchRequest(ctx, userA, userC, nil)
	if err != nil {
		t.Fatalf("create for decline: %v", err)
	}

	// Requester can't decline
	err = svc.DeclineMatchRequest(ctx, req2.ID, userA)
	assertServiceError(t, err, 403, "errors.not_recipient")

	// Recipient declines
	err = svc.DeclineMatchRequest(ctx, req2.ID, userC)
	if err != nil {
		t.Fatalf("decline: %v", err)
	}

	// Can't decline again
	err = svc.DeclineMatchRequest(ctx, req2.ID, userC)
	assertServiceError(t, err, 409, "errors.not_pending")

	// Cancel flow
	req3, err := svc.CreateMatchRequest(ctx, userC, userA, nil)
	if err != nil {
		t.Fatalf("create for cancel: %v", err)
	}

	// Recipient can't cancel
	err = svc.CancelMatchRequest(ctx, req3.ID, userA)
	assertServiceError(t, err, 403, "errors.not_requester")

	// Requester cancels
	err = svc.CancelMatchRequest(ctx, req3.ID, userC)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

func TestConnectionService_ListAndGet(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewConnectionRepository(pool)
	svc := NewConnectionService(repo, 5, 500)

	userA := createConnTestUser(t, ctx, pool, "list-a@test.com", "lista")
	userB := createConnTestUser(t, ctx, pool, "list-b@test.com", "listb")

	req, err := svc.CreateMatchRequest(ctx, userA, userB, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// List incoming for userB
	result, err := svc.ListMatchRequests(ctx, userB, "incoming", "pending", nil, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != req.ID {
		t.Errorf("wrong request ID")
	}

	// List outgoing for userA
	result, err = svc.ListMatchRequests(ctx, userA, "outgoing", "pending", nil, 20)
	if err != nil {
		t.Fatalf("list outgoing: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	// Get single request
	got, err := svc.GetMatchRequest(ctx, req.ID, userA)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != req.ID {
		t.Errorf("wrong request ID")
	}

	// Non-participant can't get
	userC := createConnTestUser(t, ctx, pool, "list-c@test.com", "listc")
	_, err = svc.GetMatchRequest(ctx, req.ID, userC)
	assertServiceError(t, err, 403, "errors.not_participant")
}

func TestConnectionService_PreAcceptMessages(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewConnectionRepository(pool)
	svc := NewConnectionService(repo, 2, 500) // limit=2 for testing

	userA := createConnTestUser(t, ctx, pool, "msg-a@test.com", "msga")
	userB := createConnTestUser(t, ctx, pool, "msg-b@test.com", "msgb")

	req, err := svc.CreateMatchRequest(ctx, userA, userB, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Send messages
	m1, err := svc.CreatePreAcceptMessage(ctx, req.ID, userA, "Hello!")
	if err != nil {
		t.Fatalf("send msg 1: %v", err)
	}
	if m1.Body != "Hello!" {
		t.Errorf("wrong body: %s", m1.Body)
	}

	_, err = svc.CreatePreAcceptMessage(ctx, req.ID, userA, "Second message")
	if err != nil {
		t.Fatalf("send msg 2: %v", err)
	}

	// Hit limit
	_, err = svc.CreatePreAcceptMessage(ctx, req.ID, userA, "Too many")
	assertServiceError(t, err, 429, "errors.message_limit")

	// Recipient can still send (separate limit)
	_, err = svc.CreatePreAcceptMessage(ctx, req.ID, userB, "Reply!")
	if err != nil {
		t.Fatalf("recipient send: %v", err)
	}

	// List messages
	msgs, err := svc.ListPreAcceptMessages(ctx, req.ID, userA, nil, 20)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs.Items) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs.Items))
	}

	// Empty body
	_, err = svc.CreatePreAcceptMessage(ctx, req.ID, userB, "")
	assertServiceError(t, err, 400, "errors.message_required")

	// Non-participant
	userC := createConnTestUser(t, ctx, pool, "msg-c@test.com", "msgc")
	_, err = svc.CreatePreAcceptMessage(ctx, req.ID, userC, "Intruder!")
	assertServiceError(t, err, 403, "errors.not_participant")
}

func TestConnectionService_BlockedUsers(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewConnectionRepository(pool)
	svc := NewConnectionService(repo, 5, 500)

	userA := createConnTestUser(t, ctx, pool, "block-a@test.com", "blocka")
	userB := createConnTestUser(t, ctx, pool, "block-b@test.com", "blockb")

	// Block userB
	_, err := pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1, $2)`, userA, userB)
	if err != nil {
		t.Fatalf("block: %v", err)
	}

	_, err = svc.CreateMatchRequest(ctx, userA, userB, nil)
	assertServiceError(t, err, 403, "errors.user_blocked")

	// Reverse direction also blocked
	_, err = svc.CreateMatchRequest(ctx, userB, userA, nil)
	assertServiceError(t, err, 403, "errors.user_blocked")
}

// --- Helpers ---

func createConnTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email, handle string) string {
	t.Helper()
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable, birth_year, birth_month)
		VALUES ($1, $2, $2, 'UTC', true, 2000, 6)`, userID, handle); err != nil {
		t.Fatalf("create profile %s: %v", handle, err)
	}
	return userID
}

func assertServiceError(t *testing.T, err error, status int, key string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %d/%s, got nil", status, key)
	}
	svcErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *service.Error, got %T: %v", err, err)
	}
	if svcErr.Status != status || svcErr.Key != key {
		t.Errorf("expected %d/%s, got %d/%s", status, key, svcErr.Status, svcErr.Key)
	}
}

func TestAgeFromBirthYear(t *testing.T) {
	// nil year
	if age := ageFromBirthYear(nil, nil); age != nil {
		t.Errorf("expected nil, got %d", *age)
	}

	// year only
	year := 2000
	age := ageFromBirthYear(&year, nil)
	if age == nil || *age < 25 {
		t.Errorf("expected age >= 25, got %v", age)
	}

	// year + month (past month)
	m := int16(1) // January
	age = ageFromBirthYear(&year, &m)
	if age == nil || *age < 25 {
		t.Errorf("expected age >= 25 with month, got %v", age)
	}

	// year + month (future month)
	m2 := int16(12) // December
	age = ageFromBirthYear(&year, &m2)
	if age == nil {
		t.Error("expected non-nil age")
	}
}

func TestNewConnectionService_Defaults(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewConnectionRepository(pool)

	// Zero values should use defaults
	svc := NewConnectionService(repo, 0, 0)
	if svc.msgLimit != defaultPreMatchMessageLimit {
		t.Errorf("expected default msg limit %d, got %d", defaultPreMatchMessageLimit, svc.msgLimit)
	}
	if svc.msgMaxLen != defaultMatchRequestMessageMaxLen {
		t.Errorf("expected default msg max len %d, got %d", defaultMatchRequestMessageMaxLen, svc.msgMaxLen)
	}
}
