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

func TestProfileServiceOrderNormalization(t *testing.T) {
	pool := openTestPool(t)

	repo := repository.NewProfileRepository(pool)
	svc := NewProfileService(repo)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('svc-profile-order@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "OrderTester", Timezone: "UTC"})
	require.NoError(t, err)

	languages := []model.Language{
		{LanguageCode: "en", Level: 5, IsNative: true, SortOrder: 0},
		{LanguageCode: "ES", Level: 2, IsTarget: true, SortOrder: 3},
	}
	updatedLanguages, err := svc.UpdateLanguages(context.Background(), userID, languages)
	require.NoError(t, err)
	require.Len(t, updatedLanguages, 2)
	require.Equal(t, 1, updatedLanguages[0].SortOrder)
	require.Equal(t, 3, updatedLanguages[1].SortOrder)
	require.Equal(t, "es", updatedLanguages[1].LanguageCode)

	slots := []model.AvailabilitySlot{
		{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC", SortOrder: 2},
		{Weekday: 2, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC", SortOrder: 0},
		{Weekday: 3, StartLocalTime: "11:00", EndLocalTime: "12:00", Timezone: "UTC", SortOrder: 0},
	}
	updatedAvailability, err := svc.UpdateAvailability(context.Background(), userID, slots)
	require.NoError(t, err)
	require.Len(t, updatedAvailability, 3)
	require.Equal(t, 1, updatedAvailability[0].SortOrder)
	require.Equal(t, 2, updatedAvailability[1].SortOrder)
	require.Equal(t, 2, updatedAvailability[2].SortOrder)
}
