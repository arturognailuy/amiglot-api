package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

const (
	defaultPreMatchMessageLimit      = 5
	defaultMatchRequestMessageMaxLen = 500
	defaultConnectionLimit           = 20
	maxConnectionLimit               = 50
)

// ConnectionService handles connection (handshake) business logic.
type ConnectionService struct {
	repo       *repository.ConnectionRepository
	msgLimit   int
	msgMaxLen  int
}

// NewConnectionService creates a new ConnectionService.
func NewConnectionService(repo *repository.ConnectionRepository, msgLimit, msgMaxLen int) *ConnectionService {
	if msgLimit <= 0 {
		msgLimit = defaultPreMatchMessageLimit
	}
	if msgMaxLen <= 0 {
		msgMaxLen = defaultMatchRequestMessageMaxLen
	}
	return &ConnectionService{repo: repo, msgLimit: msgLimit, msgMaxLen: msgMaxLen}
}

// CreateMatchRequestResult is the result of creating a match request.
type CreateMatchRequestResult struct {
	ID             string
	RequesterID    string
	RecipientID    string
	Status         string
	InitialMessage *string
	CreatedAt      time.Time
}

// CreateMatchRequest validates preconditions and creates a match request.
func (s *ConnectionService) CreateMatchRequest(ctx context.Context, requesterID, recipientID string, initialMessage *string) (*CreateMatchRequestResult, error) {
	// Self-request check
	if requesterID == recipientID {
		return nil, &Error{Status: 400, Key: "errors.self_request"}
	}

	// Validate initial message length
	if initialMessage != nil && len(*initialMessage) > s.msgMaxLen {
		return nil, &Error{Status: 400, Key: "errors.message_too_long"}
	}

	// Recipient must exist and be discoverable
	discoverable, err := s.repo.IsDiscoverable(ctx, recipientID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if !discoverable {
		return nil, &Error{Status: 404, Key: "errors.user_not_found"}
	}

	// Check if either user blocked the other
	blocked, err := s.repo.CheckUserBlocked(ctx, requesterID, recipientID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if blocked {
		return nil, &Error{Status: 403, Key: "errors.user_blocked"}
	}

	// Check for existing pending request
	pending, err := s.repo.CheckPendingRequest(ctx, requesterID, recipientID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if pending {
		return nil, &Error{Status: 409, Key: "errors.request_exists"}
	}

	// Check for existing active match
	matched, err := s.repo.CheckExistingMatch(ctx, requesterID, recipientID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if matched {
		return nil, &Error{Status: 409, Key: "errors.already_matched"}
	}

	id, createdAt, _, _, err := s.repo.CreateMatchRequest(ctx, requesterID, recipientID, initialMessage)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	return &CreateMatchRequestResult{
		ID:             id,
		RequesterID:    requesterID,
		RecipientID:    recipientID,
		Status:         "pending",
		InitialMessage: initialMessage,
		CreatedAt:      createdAt,
	}, nil
}

// MatchRequestListItem represents a match request in a list.
type MatchRequestListItem struct {
	ID                  string
	RequesterID         string
	RecipientID         string
	RequesterHandle     string
	RequesterCountry    *string
	RequesterAge        *int
	RecipientHandle     string
	RecipientCountry    *string
	RecipientAge        *int
	Status              string
	MessageCount        int
	LastMessageAt       *time.Time
	CreatedAt           time.Time
}

// MatchRequestListResult is the result of listing match requests.
type MatchRequestListResult struct {
	Items      []MatchRequestListItem
	NextCursor *string
}

// ListMatchRequests returns paginated match requests for a user.
func (s *ConnectionService) ListMatchRequests(ctx context.Context, userID, direction, status string, cursor *string, limit int) (*MatchRequestListResult, error) {
	if direction != "incoming" && direction != "outgoing" {
		direction = "incoming"
	}
	if status == "" {
		status = "pending"
	}
	if limit <= 0 || limit > maxConnectionLimit {
		limit = defaultConnectionLimit
	}

	rows, err := s.repo.ListMatchRequests(ctx, userID, direction, status, cursor, limit+1)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	var nextCursor *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1].ID
		nextCursor = &last
	}

	items := make([]MatchRequestListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, MatchRequestListItem{
			ID:               r.ID,
			RequesterID:      r.RequesterID,
			RecipientID:      r.RecipientID,
			RequesterHandle:  r.RequesterHandle,
			RequesterCountry: r.RequesterCountry,
			RecipientHandle:  r.RecipientHandle,
			RecipientCountry: r.RecipientCountry,
			RequesterAge:     ageFromBirthYear(r.RequesterBirthYear, r.RequesterBirthMonth),
			RecipientAge:     ageFromBirthYear(r.RecipientBirthYear, r.RecipientBirthMonth),
			Status:           r.Status,
			MessageCount:     r.MessageCount,
			LastMessageAt:    r.LastMessageAt,
			CreatedAt:        r.CreatedAt,
		})
	}

	return &MatchRequestListResult{Items: items, NextCursor: nextCursor}, nil
}

// GetMatchRequestResult is the result of getting a single match request.
type GetMatchRequestResult struct {
	MatchRequestListItem
}

// GetMatchRequest returns a single match request. Caller must be requester or recipient.
func (s *ConnectionService) GetMatchRequest(ctx context.Context, requestID, userID string) (*GetMatchRequestResult, error) {
	row, err := s.repo.GetMatchRequest(ctx, requestID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if row == nil {
		return nil, &Error{Status: 404, Key: "errors.request_not_found"}
	}

	if row.RequesterID != userID && row.RecipientID != userID {
		return nil, &Error{Status: 403, Key: "errors.not_participant"}
	}

	return &GetMatchRequestResult{
		MatchRequestListItem: MatchRequestListItem{
			ID:               row.ID,
			RequesterID:      row.RequesterID,
			RecipientID:      row.RecipientID,
			RequesterHandle:  row.RequesterHandle,
			RequesterCountry: row.RequesterCountry,
			RecipientHandle:  row.RecipientHandle,
			RecipientCountry: row.RecipientCountry,
			RequesterAge:     ageFromBirthYear(row.RequesterBirthYear, row.RequesterBirthMonth),
			RecipientAge:     ageFromBirthYear(row.RecipientBirthYear, row.RecipientBirthMonth),
			Status:           row.Status,
			MessageCount:     row.MessageCount,
			LastMessageAt:    row.LastMessageAt,
			CreatedAt:        row.CreatedAt,
		},
	}, nil
}

// AcceptMatchRequest accepts a pending request. Only the recipient can accept.
func (s *ConnectionService) AcceptMatchRequest(ctx context.Context, requestID, userID string) (string, error) {
	status, _, recipientID, err := s.repo.GetMatchRequestStatus(ctx, requestID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", &Error{Status: 404, Key: "errors.request_not_found"}
		}
		return "", &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	if recipientID != userID {
		return "", &Error{Status: 403, Key: "errors.not_recipient"}
	}
	if status != "pending" {
		return "", &Error{Status: 409, Key: "errors.not_pending"}
	}

	matchID, err := s.repo.AcceptMatchRequest(ctx, requestID, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", &Error{Status: 409, Key: "errors.already_matched"}
		}
		return "", &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	return matchID, nil
}

// DeclineMatchRequest declines a pending request. Only the recipient can decline.
func (s *ConnectionService) DeclineMatchRequest(ctx context.Context, requestID, userID string) error {
	status, _, recipientID, err := s.repo.GetMatchRequestStatus(ctx, requestID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &Error{Status: 404, Key: "errors.request_not_found"}
		}
		return &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	if recipientID != userID {
		return &Error{Status: 403, Key: "errors.not_recipient"}
	}
	if status != "pending" {
		return &Error{Status: 409, Key: "errors.not_pending"}
	}

	if err := s.repo.DeclineMatchRequest(ctx, requestID, userID); err != nil {
		return &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	return nil
}

// CancelMatchRequest cancels a pending request. Only the requester can cancel.
func (s *ConnectionService) CancelMatchRequest(ctx context.Context, requestID, userID string) error {
	status, requesterID, _, err := s.repo.GetMatchRequestStatus(ctx, requestID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &Error{Status: 404, Key: "errors.request_not_found"}
		}
		return &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	if requesterID != userID {
		return &Error{Status: 403, Key: "errors.not_requester"}
	}
	if status != "pending" {
		return &Error{Status: 409, Key: "errors.not_pending"}
	}

	if err := s.repo.CancelMatchRequest(ctx, requestID, userID); err != nil {
		return &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	return nil
}

// PreAcceptMessage represents a pre-accept message.
type PreAcceptMessage struct {
	ID        string
	SenderID  string
	Body      string
	CreatedAt time.Time
}

// PreAcceptMessageListResult is the result of listing pre-accept messages.
type PreAcceptMessageListResult struct {
	Items      []PreAcceptMessage
	NextCursor *string
}

// ListPreAcceptMessages returns paginated pre-accept messages for a request.
func (s *ConnectionService) ListPreAcceptMessages(ctx context.Context, requestID, userID string, cursor *string, limit int) (*PreAcceptMessageListResult, error) {
	// Verify participation
	_, requesterID, recipientID, err := s.repo.GetMatchRequestStatus(ctx, requestID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &Error{Status: 404, Key: "errors.request_not_found"}
		}
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if requesterID != userID && recipientID != userID {
		return nil, &Error{Status: 403, Key: "errors.not_participant"}
	}

	if limit <= 0 || limit > maxConnectionLimit {
		limit = defaultConnectionLimit
	}

	rows, err := s.repo.ListPreAcceptMessages(ctx, requestID, cursor, limit+1)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	var nextCursor *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1].ID
		nextCursor = &last
	}

	items := make([]PreAcceptMessage, 0, len(rows))
	for _, r := range rows {
		items = append(items, PreAcceptMessage{
			ID:        r.ID,
			SenderID:  r.SenderID,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
		})
	}

	return &PreAcceptMessageListResult{Items: items, NextCursor: nextCursor}, nil
}

// CreatePreAcceptMessage sends a pre-accept message.
func (s *ConnectionService) CreatePreAcceptMessage(ctx context.Context, requestID, senderID, body string) (*PreAcceptMessage, error) {
	if len(body) == 0 {
		return nil, &Error{Status: 400, Key: "errors.message_required"}
	}
	if len(body) > s.msgMaxLen {
		return nil, &Error{Status: 400, Key: "errors.message_too_long"}
	}

	status, requesterID, recipientID, err := s.repo.GetMatchRequestStatus(ctx, requestID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &Error{Status: 404, Key: "errors.request_not_found"}
		}
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	if status != "pending" {
		return nil, &Error{Status: 409, Key: "errors.not_pending"}
	}
	if senderID != requesterID && senderID != recipientID {
		return nil, &Error{Status: 403, Key: "errors.not_participant"}
	}

	// Check message limit
	count, err := s.repo.CountPreAcceptMessages(ctx, requestID, senderID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if count >= s.msgLimit {
		return nil, &Error{Status: 429, Key: "errors.message_limit"}
	}

	id, createdAt, err := s.repo.CreatePreAcceptMessage(ctx, requestID, senderID, body)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	return &PreAcceptMessage{
		ID:        id,
		SenderID:  senderID,
		Body:      body,
		CreatedAt: createdAt,
	}, nil
}

// ageFromBirthYear computes approximate age from birth year and optional birth month.
func ageFromBirthYear(year *int, month *int16) *int {
	if year == nil {
		return nil
	}
	now := time.Now()
	age := now.Year() - *year
	if month != nil && int(now.Month()) < int(*month) {
		age--
	}
	return &age
}
