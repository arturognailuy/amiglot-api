package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectionRepository handles match request and pre-accept messaging queries.
type ConnectionRepository struct {
	pool *pgxpool.Pool
}

// NewConnectionRepository creates a new ConnectionRepository.
func NewConnectionRepository(pool *pgxpool.Pool) *ConnectionRepository {
	return &ConnectionRepository{pool: pool}
}

// Pool returns the underlying connection pool.
func (r *ConnectionRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// MatchRequestRow represents a match request row.
type MatchRequestRow struct {
	ID                  string
	RequesterID         string
	RecipientID         string
	Status              string
	CreatedAt           time.Time
	RespondedAt         *time.Time
	RequesterHandle     string
	RequesterCountry    *string
	RequesterBirthYear  *int
	RequesterBirthMonth *int16
	RecipientHandle     string
	RecipientCountry    *string
	RecipientBirthYear  *int
	RecipientBirthMonth *int16
	MessageCount        int
	LastMessageAt       *time.Time
}

// MessageRow represents a pre-accept message row.
type MessageRow struct {
	ID        string
	SenderID  string
	Body      string
	CreatedAt time.Time
}

// CreateMatchRequest inserts a new match request and optionally an initial message.
// Returns the request ID and created_at.
func (r *ConnectionRepository) CreateMatchRequest(ctx context.Context, requesterID, recipientID string, initialMessage *string) (string, time.Time, *string, *time.Time, error) {
	var id string
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO match_requests (requester_id, recipient_id, status)
		 VALUES ($1, $2, 'pending')
		 RETURNING id, created_at`,
		requesterID, recipientID,
	).Scan(&id, &createdAt)
	if err != nil {
		return "", time.Time{}, nil, nil, err
	}

	var msgID *string
	var msgCreatedAt *time.Time
	if initialMessage != nil && *initialMessage != "" {
		var mid string
		var mca time.Time
		err = r.pool.QueryRow(ctx,
			`INSERT INTO messages (match_request_id, sender_id, body)
			 VALUES ($1, $2, $3)
			 RETURNING id, created_at`,
			id, requesterID, *initialMessage,
		).Scan(&mid, &mca)
		if err != nil {
			return "", time.Time{}, nil, nil, err
		}
		msgID = &mid
		msgCreatedAt = &mca
	}

	return id, createdAt, msgID, msgCreatedAt, nil
}

const listMatchRequestsSQL = `
SELECT
    mr.id, mr.status, mr.created_at,
    mr.requester_id, mr.recipient_id,
    rp.handle AS requester_handle,
    rp.country_code AS requester_country,
    rp.birth_year AS requester_birth_year,
    rp.birth_month AS requester_birth_month,
    pp.handle AS recipient_handle,
    pp.country_code AS recipient_country,
    pp.birth_year AS recipient_birth_year,
    pp.birth_month AS recipient_birth_month,
    (SELECT COUNT(*) FROM messages m WHERE m.match_request_id = mr.id) AS message_count,
    (SELECT MAX(m.created_at) FROM messages m WHERE m.match_request_id = mr.id) AS last_message_at
FROM match_requests mr
JOIN profiles rp ON rp.user_id = mr.requester_id
JOIN profiles pp ON pp.user_id = mr.recipient_id
WHERE CASE WHEN $1 = 'incoming' THEN mr.recipient_id = $2 ELSE mr.requester_id = $2 END
  AND ($3 = 'all' OR mr.status = $3)
  AND ($4::uuid IS NULL OR mr.created_at < (SELECT created_at FROM match_requests WHERE id = $4))
ORDER BY mr.created_at DESC
LIMIT $5
`

// ListMatchRequests returns paginated match requests for a user.
func (r *ConnectionRepository) ListMatchRequests(ctx context.Context, userID, direction, status string, cursor *string, limit int) ([]MatchRequestRow, error) {
	rows, err := r.pool.Query(ctx, listMatchRequestsSQL, direction, userID, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MatchRequestRow
	for rows.Next() {
		var m MatchRequestRow
		if err := rows.Scan(
			&m.ID, &m.Status, &m.CreatedAt,
			&m.RequesterID, &m.RecipientID,
			&m.RequesterHandle, &m.RequesterCountry, &m.RequesterBirthYear, &m.RequesterBirthMonth,
			&m.RecipientHandle, &m.RecipientCountry, &m.RecipientBirthYear, &m.RecipientBirthMonth,
			&m.MessageCount, &m.LastMessageAt,
		); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

const getMatchRequestSQL = `
SELECT
    mr.id, mr.status, mr.created_at,
    mr.requester_id, mr.recipient_id,
    rp.handle AS requester_handle,
    rp.country_code AS requester_country,
    rp.birth_year AS requester_birth_year,
    rp.birth_month AS requester_birth_month,
    pp.handle AS recipient_handle,
    pp.country_code AS recipient_country,
    pp.birth_year AS recipient_birth_year,
    pp.birth_month AS recipient_birth_month,
    (SELECT COUNT(*) FROM messages m WHERE m.match_request_id = mr.id) AS message_count,
    (SELECT MAX(m.created_at) FROM messages m WHERE m.match_request_id = mr.id) AS last_message_at
FROM match_requests mr
JOIN profiles rp ON rp.user_id = mr.requester_id
JOIN profiles pp ON pp.user_id = mr.recipient_id
WHERE mr.id = $1
`

// GetMatchRequest returns a single match request. Returns nil if not found.
func (r *ConnectionRepository) GetMatchRequest(ctx context.Context, requestID string) (*MatchRequestRow, error) {
	var m MatchRequestRow
	err := r.pool.QueryRow(ctx, getMatchRequestSQL, requestID).Scan(
		&m.ID, &m.Status, &m.CreatedAt,
		&m.RequesterID, &m.RecipientID,
		&m.RequesterHandle, &m.RequesterCountry, &m.RequesterBirthYear, &m.RequesterBirthMonth,
		&m.RecipientHandle, &m.RecipientCountry, &m.RecipientBirthYear, &m.RecipientBirthMonth,
		&m.MessageCount, &m.LastMessageAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// AcceptMatchRequest performs the accept transaction: update request, create match, re-associate messages.
// Returns the new match ID.
func (r *ConnectionRepository) AcceptMatchRequest(ctx context.Context, requestID, recipientID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	// Step 1: Lock and update the request
	var requesterID, recID string
	err = tx.QueryRow(ctx,
		`UPDATE match_requests
		 SET status = 'accepted', responded_at = now()
		 WHERE id = $1 AND recipient_id = $2 AND status = 'pending'
		 RETURNING requester_id, recipient_id`,
		requestID, recipientID,
	).Scan(&requesterID, &recID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", err
		}
		return "", err
	}

	// Step 2: Create match (LEAST/GREATEST for unique index)
	var matchID string
	err = tx.QueryRow(ctx,
		`INSERT INTO matches (user_a, user_b)
		 VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		requesterID, recID,
	).Scan(&matchID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Match already exists (race condition)
			return "", pgx.ErrNoRows
		}
		return "", err
	}

	// Step 3: Re-associate messages
	_, err = tx.Exec(ctx,
		`UPDATE messages
		 SET match_id = $1, match_request_id = NULL
		 WHERE match_request_id = $2`,
		matchID, requestID,
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return matchID, nil
}

// DeclineMatchRequest declines a pending request (only recipient).
func (r *ConnectionRepository) DeclineMatchRequest(ctx context.Context, requestID, recipientID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE match_requests
		 SET status = 'declined', responded_at = now()
		 WHERE id = $1 AND recipient_id = $2 AND status = 'pending'`,
		requestID, recipientID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CancelMatchRequest cancels a pending request (only requester).
func (r *ConnectionRepository) CancelMatchRequest(ctx context.Context, requestID, requesterID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE match_requests
		 SET status = 'canceled', responded_at = now()
		 WHERE id = $1 AND requester_id = $2 AND status = 'pending'`,
		requestID, requesterID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CreatePreAcceptMessage inserts a pre-accept message.
func (r *ConnectionRepository) CreatePreAcceptMessage(ctx context.Context, matchRequestID, senderID, body string) (string, time.Time, error) {
	var id string
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO messages (match_request_id, sender_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		matchRequestID, senderID, body,
	).Scan(&id, &createdAt)
	return id, createdAt, err
}

// ListPreAcceptMessages returns paginated messages for a match request.
func (r *ConnectionRepository) ListPreAcceptMessages(ctx context.Context, matchRequestID string, cursor *string, limit int) ([]MessageRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, sender_id, body, created_at
		 FROM messages
		 WHERE match_request_id = $1
		   AND ($2::uuid IS NULL OR created_at < (SELECT created_at FROM messages WHERE id = $2))
		 ORDER BY created_at DESC
		 LIMIT $3`,
		matchRequestID, cursor, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MessageRow
	for rows.Next() {
		var m MessageRow
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// CountPreAcceptMessages counts messages sent by a specific user for a match request.
func (r *ConnectionRepository) CountPreAcceptMessages(ctx context.Context, matchRequestID, senderID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages WHERE match_request_id = $1 AND sender_id = $2`,
		matchRequestID, senderID,
	).Scan(&count)
	return count, err
}

// CheckPendingRequest checks if a pending request exists between two users (either direction).
func (r *ConnectionRepository) CheckPendingRequest(ctx context.Context, userA, userB string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM match_requests
			WHERE status = 'pending'
			  AND ((requester_id = $1 AND recipient_id = $2) OR (requester_id = $2 AND recipient_id = $1))
		)`, userA, userB,
	).Scan(&exists)
	return exists, err
}

// CheckExistingMatch checks if an active match exists between two users.
func (r *ConnectionRepository) CheckExistingMatch(ctx context.Context, userA, userB string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM matches
			WHERE LEAST(user_a, user_b) = LEAST($1::uuid, $2::uuid)
			  AND GREATEST(user_a, user_b) = GREATEST($1::uuid, $2::uuid)
			  AND closed_at IS NULL
		)`, userA, userB,
	).Scan(&exists)
	return exists, err
}

// CheckUserBlocked checks if either user has blocked the other.
func (r *ConnectionRepository) CheckUserBlocked(ctx context.Context, userA, userB string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
		)`, userA, userB,
	).Scan(&exists)
	return exists, err
}

// IsDiscoverable checks if a user's profile exists and is discoverable.
func (r *ConnectionRepository) IsDiscoverable(ctx context.Context, userID string) (bool, error) {
	var discoverable bool
	err := r.pool.QueryRow(ctx, `SELECT discoverable FROM profiles WHERE user_id = $1`, userID).Scan(&discoverable)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return discoverable, nil
}

// GetMatchRequestStatus returns the status, requester_id, and recipient_id for a match request.
func (r *ConnectionRepository) GetMatchRequestStatus(ctx context.Context, requestID string) (string, string, string, error) {
	var status, requesterID, recipientID string
	err := r.pool.QueryRow(ctx,
		`SELECT status, requester_id, recipient_id FROM match_requests WHERE id = $1`,
		requestID,
	).Scan(&status, &requesterID, &recipientID)
	return status, requesterID, recipientID, err
}
