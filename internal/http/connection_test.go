package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createConnUser(t *testing.T, pool *pgxpool.Pool, email, handle string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = pool.Exec(context.Background(),
		`INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		 VALUES ($1, $2, $2, 'UTC', true)`, id, handle)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	return id
}

func TestConnectionEndpoint_CreateRequest(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "conn-http-a@test.com", "conhttpa")
	userB := createConnUser(t, pool, "conn-http-b@test.com", "conhttpb")

	// Create a request
	body := fmt.Sprintf(`{"recipient_id":"%s","initial_message":"Hi!"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result struct {
		ID          string `json:"id"`
		RequesterID string `json:"requester_id"`
		RecipientID string `json:"recipient_id"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != "pending" {
		t.Errorf("expected pending, got %s", result.Status)
	}

	// Self-request
	selfBody := fmt.Sprintf(`{"recipient_id":"%s"}`, userA)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(selfBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Id", userA)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for self-request, got %d", rec2.Code)
	}

	// Duplicate request
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-User-Id", userA)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d", rec3.Code)
	}
}

func TestConnectionEndpoint_ListRequests(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "list-http-a@test.com", "lsthttpa")
	userB := createConnUser(t, pool, "list-http-b@test.com", "lsthttpb")

	// Create a request first
	body := fmt.Sprintf(`{"recipient_id":"%s"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d: %s", rec.Code, rec.Body.String())
	}

	// List incoming for userB
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/match-requests?direction=incoming&status=pending", nil)
	req2.Header.Set("X-User-Id", userB)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list: %d: %s", rec2.Code, rec2.Body.String())
	}

	var listResult struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(rec2.Body).Decode(&listResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(listResult.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(listResult.Items))
	}
}

func TestConnectionEndpoint_AcceptFlow(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "acc-http-a@test.com", "acchttpa")
	userB := createConnUser(t, pool, "acc-http-b@test.com", "acchttpb")

	// Create request
	body := fmt.Sprintf(`{"recipient_id":"%s","initial_message":"Hey!"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Accept
	acceptReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/match-requests/%s/accept", created.ID), nil)
	acceptReq.Header.Set("X-User-Id", userB)
	acceptRec := httptest.NewRecorder()
	mux.ServeHTTP(acceptRec, acceptReq)

	if acceptRec.Code != http.StatusOK {
		t.Fatalf("accept: %d: %s", acceptRec.Code, acceptRec.Body.String())
	}

	var acceptResult struct {
		Ok      bool   `json:"ok"`
		MatchID string `json:"match_id"`
	}
	if err := json.NewDecoder(acceptRec.Body).Decode(&acceptResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !acceptResult.Ok || acceptResult.MatchID == "" {
		t.Errorf("expected ok with match_id, got %+v", acceptResult)
	}
}

func TestConnectionEndpoint_DeclineFlow(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "dec-http-a@test.com", "dechttpa")
	userB := createConnUser(t, pool, "dec-http-b@test.com", "dechttpb")

	body := fmt.Sprintf(`{"recipient_id":"%s"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Decline
	decReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/match-requests/%s/decline", created.ID), nil)
	decReq.Header.Set("X-User-Id", userB)
	decRec := httptest.NewRecorder()
	mux.ServeHTTP(decRec, decReq)

	if decRec.Code != http.StatusOK {
		t.Fatalf("decline: %d: %s", decRec.Code, decRec.Body.String())
	}
}

func TestConnectionEndpoint_CancelFlow(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "can-http-a@test.com", "canhttpa")
	userB := createConnUser(t, pool, "can-http-b@test.com", "canhttpb")

	body := fmt.Sprintf(`{"recipient_id":"%s"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Cancel
	canReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/match-requests/%s/cancel", created.ID), nil)
	canReq.Header.Set("X-User-Id", userA)
	canRec := httptest.NewRecorder()
	mux.ServeHTTP(canRec, canReq)

	if canRec.Code != http.StatusOK {
		t.Fatalf("cancel: %d: %s", canRec.Code, canRec.Body.String())
	}
}

func TestConnectionEndpoint_Messages(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "msg-http-a@test.com", "msghttpa")
	userB := createConnUser(t, pool, "msg-http-b@test.com", "msghttpb")

	// Create request
	body := fmt.Sprintf(`{"recipient_id":"%s"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Send message
	msgBody := `{"body":"Hello from test!"}`
	msgReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/match-requests/%s/messages", created.ID), bytes.NewBufferString(msgBody))
	msgReq.Header.Set("Content-Type", "application/json")
	msgReq.Header.Set("X-User-Id", userA)
	msgRec := httptest.NewRecorder()
	mux.ServeHTTP(msgRec, msgReq)

	if msgRec.Code != http.StatusOK {
		t.Fatalf("send msg: %d: %s", msgRec.Code, msgRec.Body.String())
	}

	// List messages
	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/match-requests/%s/messages", created.ID), nil)
	listReq.Header.Set("X-User-Id", userA)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list msg: %d: %s", listRec.Code, listRec.Body.String())
	}

	var msgList struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&msgList); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(msgList.Items) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgList.Items))
	}

	// Send empty message — should fail
	emptyMsg := `{"body":""}`
	emptyReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/match-requests/%s/messages", created.ID), bytes.NewBufferString(emptyMsg))
	emptyReq.Header.Set("Content-Type", "application/json")
	emptyReq.Header.Set("X-User-Id", userA)
	emptyRec := httptest.NewRecorder()
	mux.ServeHTTP(emptyRec, emptyReq)
	if emptyRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty message, got %d", emptyRec.Code)
	}
}

func TestConnectionEndpoint_GetRequest(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createConnUser(t, pool, "get-http-a@test.com", "gethttpa")
	userB := createConnUser(t, pool, "get-http-b@test.com", "gethttpb")

	// Create request with initial message to have LastMessageAt populated
	body := fmt.Sprintf(`{"recipient_id":"%s","initial_message":"Hi there!"}`, userB)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Get single request
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/match-requests/%s", created.ID), nil)
	getReq.Header.Set("X-User-Id", userA)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get: %d: %s", getRec.Code, getRec.Body.String())
	}

	// Non-participant
	userC := createConnUser(t, pool, "get-http-c@test.com", "gethttpc")
	getReq2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/match-requests/%s", created.ID), nil)
	getReq2.Header.Set("X-User-Id", userC)
	getRec2 := httptest.NewRecorder()
	mux.ServeHTTP(getRec2, getReq2)

	if getRec2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-participant, got %d", getRec2.Code)
	}
}
