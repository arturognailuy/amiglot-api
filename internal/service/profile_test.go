package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/model"
	"github.com/gnailuy/amiglot-api/internal/repository"
)

func TestProfileServiceFlow(t *testing.T) {
	pool := openTestPool(t)

	repo := repository.NewProfileRepository(pool)
	svc := NewProfileService(repo)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('svc-profile@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{})
	require.Error(t, err)

	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "Arturo", Timezone: "UTC"})
	require.NoError(t, err)

	languages := []model.Language{{LanguageCode: "en", Level: 5, IsNative: true, SortOrder: 1}}
	updatedLanguages, err := svc.UpdateLanguages(context.Background(), userID, languages)
	require.NoError(t, err)
	require.Len(t, updatedLanguages, 1)

	availability := []model.AvailabilitySlot{{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "", SortOrder: 1}}
	updatedAvailability, err := svc.UpdateAvailability(context.Background(), userID, availability)
	require.NoError(t, err)
	require.Equal(t, "UTC", updatedAvailability[0].Timezone)

	user, profile, gotLanguages, gotAvailability, err := svc.GetProfile(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, userID, user.ID)
	require.Equal(t, "arturo", profile.Handle)
	require.True(t, profile.Discoverable)
	require.Len(t, gotLanguages, 1)
	require.Len(t, gotAvailability, 1)
}

func TestProfileServiceValidation(t *testing.T) {
	pool := openTestPool(t)

	repo := repository.NewProfileRepository(pool)
	svc := NewProfileService(repo)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('svc-profile2@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{{LanguageCode: "en", Level: 1, IsNative: false}})
	require.Error(t, err)

	_, err = svc.CheckHandleAvailability(context.Background(), "", "handle")
	require.Error(t, err)
}
