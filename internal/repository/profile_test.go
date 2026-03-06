package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/model"
)

func TestProfileRepositoryFlow(t *testing.T) {
	pool := openTestPool(t)

	repo := NewProfileRepository(pool)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('repo-profile@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	user, err := repo.LoadUser(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, userID, user.ID)
	require.Equal(t, "repo-profile@example.com", user.Email)

	profile := model.Profile{
		Handle:      "tester",
		BirthYear:   nil,
		BirthMonth:  nil,
		CountryCode: nil,
		Timezone:    "UTC",
	}
	err = repo.UpsertProfile(context.Background(), userID, profile)
	require.NoError(t, err)

	loadedProfile, err := repo.LoadProfile(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, "tester", loadedProfile.Handle)
	require.Equal(t, "UTC", loadedProfile.Timezone)

	languages := []model.Language{{LanguageCode: "en", Level: 5, IsNative: true, SortOrder: 1}}
	err = repo.ReplaceLanguages(context.Background(), userID, languages)
	require.NoError(t, err)

	hasNative, err := repo.HasNativeLanguage(context.Background(), userID)
	require.NoError(t, err)
	require.True(t, hasNative)

	availability := []model.AvailabilitySlot{{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC", SortOrder: 1}}
	err = repo.ReplaceAvailability(context.Background(), userID, availability)
	require.NoError(t, err)

	loadedAvailability, err := repo.LoadAvailability(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, loadedAvailability, 1)

	available, err := repo.CheckHandleAvailability(context.Background(), userID, "tester")
	require.NoError(t, err)
	require.True(t, available)

	_, err = pool.Exec(context.Background(), `INSERT INTO users (email) VALUES ('other@example.com')`)
	require.NoError(t, err)

	available, err = repo.CheckHandleAvailability(context.Background(), "other-user", "tester")
	require.NoError(t, err)
	require.False(t, available)

	handle, timezone, err := repo.LoadHandleAndTimezone(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, "tester", handle)
	require.Equal(t, "UTC", timezone)

	err = repo.UpdateDiscoverable(context.Background(), userID, true)
	require.NoError(t, err)

	loadedProfile, err = repo.LoadProfile(context.Background(), userID)
	require.NoError(t, err)
	require.True(t, loadedProfile.Discoverable)
}
