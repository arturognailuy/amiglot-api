package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionRepository_FullFlow(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	repo := NewConnectionRepository(pool)

	// Create two users with discoverable profiles
	var userA, userB string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('conn-repo-a@test.com') RETURNING id`).Scan(&userA)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable) VALUES ($1, 'connrepoa', 'connrepoa', 'UTC', true)`, userA)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('conn-repo-b@test.com') RETURNING id`).Scan(&userB)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable) VALUES ($1, 'connrepob', 'connrepob', 'UTC', true)`, userB)
	require.NoError(t, err)

	// Pool accessor
	require.Equal(t, pool, repo.Pool())

	// IsDiscoverable
	disc, err := repo.IsDiscoverable(ctx, userA)
	require.NoError(t, err)
	require.True(t, disc)

	disc, err = repo.IsDiscoverable(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	require.False(t, disc)

	// CheckUserBlocked — no blocks
	blocked, err := repo.CheckUserBlocked(ctx, userA, userB)
	require.NoError(t, err)
	require.False(t, blocked)

	// CheckPendingRequest — none yet
	pending, err := repo.CheckPendingRequest(ctx, userA, userB)
	require.NoError(t, err)
	require.False(t, pending)

	// CheckExistingMatch — none yet
	matched, err := repo.CheckExistingMatch(ctx, userA, userB)
	require.NoError(t, err)
	require.False(t, matched)

	// CreateMatchRequest
	msg := "Hello!"
	id, createdAt, msgID, _, err := repo.CreateMatchRequest(ctx, userA, userB, &msg)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	require.False(t, createdAt.IsZero())
	require.NotNil(t, msgID)

	// CreateMatchRequest without message
	var userC string
	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('conn-repo-c@test.com') RETURNING id`).Scan(&userC)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable) VALUES ($1, 'connrepoc', 'connrepoc', 'UTC', true)`, userC)
	require.NoError(t, err)

	id2, _, msgID2, _, err := repo.CreateMatchRequest(ctx, userA, userC, nil)
	require.NoError(t, err)
	require.NotEmpty(t, id2)
	require.Nil(t, msgID2)

	// CheckPendingRequest — now exists
	pending, err = repo.CheckPendingRequest(ctx, userA, userB)
	require.NoError(t, err)
	require.True(t, pending)

	// GetMatchRequest
	row, err := repo.GetMatchRequest(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, "pending", row.Status)

	// GetMatchRequest — not found
	row, err = repo.GetMatchRequest(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	require.Nil(t, row)

	// ListMatchRequests — incoming for userB
	rows, err := repo.ListMatchRequests(ctx, userB, "incoming", "pending", nil, 20)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	// GetMatchRequestStatus
	status, reqID, recID, err := repo.GetMatchRequestStatus(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "pending", status)
	require.Equal(t, userA, reqID)
	require.Equal(t, userB, recID)

	// CreatePreAcceptMessage
	pmID, pmCreated, err := repo.CreatePreAcceptMessage(ctx, id, userB, "Reply!")
	require.NoError(t, err)
	require.NotEmpty(t, pmID)
	require.False(t, pmCreated.IsZero())

	// CountPreAcceptMessages
	count, err := repo.CountPreAcceptMessages(ctx, id, userB)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// ListPreAcceptMessages
	msgs, err := repo.ListPreAcceptMessages(ctx, id, nil, 20)
	require.NoError(t, err)
	require.Len(t, msgs, 2) // initial + reply

	// AcceptMatchRequest
	matchID, err := repo.AcceptMatchRequest(ctx, id, userB)
	require.NoError(t, err)
	require.NotEmpty(t, matchID)

	// CheckExistingMatch — now exists
	matched, err = repo.CheckExistingMatch(ctx, userA, userB)
	require.NoError(t, err)
	require.True(t, matched)

	// DeclineMatchRequest on the second request
	err = repo.DeclineMatchRequest(ctx, id2, userC)
	require.NoError(t, err)

	// CancelMatchRequest — create and cancel
	id3, _, _, _, err := repo.CreateMatchRequest(ctx, userB, userC, nil)
	require.NoError(t, err)
	err = repo.CancelMatchRequest(ctx, id3, userB)
	require.NoError(t, err)
}
