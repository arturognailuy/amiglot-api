package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/model"
	"github.com/gnailuy/amiglot-api/internal/repository"
)

func TestError_Coverage(t *testing.T) {
	e := &Error{Status: 400, Key: "key", Err: errors.New("inner")}
	require.Equal(t, "key: inner", e.Error())
	require.Equal(t, "inner", e.Unwrap().Error())

	e2 := &Error{Status: 500, Key: "key"}
	require.Equal(t, "key", e2.Error())
	require.Nil(t, e2.Unwrap())
}

func TestProfileService_Coverage(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewProfileRepository(pool)
	svc := NewProfileService(repo)

	var userID string
	err := pool.QueryRow(context.Background(), "INSERT INTO users (email) VALUES ('svc-cov@example.com') RETURNING id").Scan(&userID)
	require.NoError(t, err)

	// CheckHandleAvailability missing user id
	_, err = svc.CheckHandleAvailability(context.Background(), "", "handle")
	require.Error(t, err)

	// CheckHandleAvailability missing handle
	_, err = svc.CheckHandleAvailability(context.Background(), userID, "")
	require.Error(t, err)

	// CheckHandleAvailability short handle
	_, err = svc.CheckHandleAvailability(context.Background(), userID, "ab")
	require.Error(t, err)

	// CheckHandleAvailability invalid chars
	_, err = svc.CheckHandleAvailability(context.Background(), userID, "ab-c")
	require.Error(t, err)

	// UpdateProfile missing user id
	_, _, _, _, err = svc.UpdateProfile(context.Background(), "", model.Profile{})
	require.Error(t, err)

	// UpdateProfile missing handle
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: ""})
	require.Error(t, err)

	// UpdateProfile short handle
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "ab"})
	require.Error(t, err)

	// UpdateProfile invalid chars
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "ab-c"})
	require.Error(t, err)

	// UpdateProfile missing timezone
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "abc"})
	require.Error(t, err)

	// UpdateProfile invalid timezone
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "abc", Timezone: "Mars/Base"})
	require.Error(t, err)

	// UpdateProfile invalid birth year
	badYear := 1800
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "abc", Timezone: "UTC", BirthYear: &badYear})
	require.Error(t, err)

	// UpdateProfile invalid birth month
	badMonth := int16(13)
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "abc", Timezone: "UTC", BirthMonth: &badMonth})
	require.Error(t, err)

	// UpdateProfile invalid country code
	badCountry := "USA"
	_, _, _, _, err = svc.UpdateProfile(context.Background(), userID, model.Profile{Handle: "abc", Timezone: "UTC", CountryCode: &badCountry})
	require.Error(t, err)

	// UpdateLanguages missing user id
	_, err = svc.UpdateLanguages(context.Background(), "", []model.Language{})
	require.Error(t, err)

	// UpdateLanguages missing languages
	_, err = svc.UpdateLanguages(context.Background(), userID, nil)
	require.Error(t, err)

	// UpdateLanguages invalid level
	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{{LanguageCode: "en", Level: 6}})
	require.Error(t, err)

	// UpdateLanguages conflict native/target
	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{{LanguageCode: "en", Level: 5, IsNative: true, IsTarget: true}})
	require.Error(t, err)

	// UpdateLanguages invalid native level
	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{{LanguageCode: "en", Level: 4, IsNative: true}})
	require.Error(t, err)

	// UpdateLanguages target level 5
	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{{LanguageCode: "en", Level: 5, IsTarget: true}})
	require.Error(t, err)

	// UpdateLanguages duplicate
	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{
		{LanguageCode: "en", Level: 5, IsNative: true},
		{LanguageCode: "en", Level: 5, IsNative: true},
	})
	require.Error(t, err)

	// UpdateLanguages no native
	_, err = svc.UpdateLanguages(context.Background(), userID, []model.Language{{LanguageCode: "es", Level: 1, IsTarget: true}})
	require.Error(t, err)

	// UpdateAvailability missing user id
	_, err = svc.UpdateAvailability(context.Background(), "", nil)
	require.Error(t, err)

	// UpdateAvailability too many slots
	slots := make([]model.AvailabilitySlot, 15)
	_, err = svc.UpdateAvailability(context.Background(), userID, slots)
	require.Error(t, err)

	// UpdateAvailability invalid weekday
	_, err = svc.UpdateAvailability(context.Background(), userID, []model.AvailabilitySlot{{Weekday: 7, StartLocalTime: "09:00", EndLocalTime: "10:00"}})
	require.Error(t, err)

	// UpdateAvailability missing time
	_, err = svc.UpdateAvailability(context.Background(), userID, []model.AvailabilitySlot{{Weekday: 1, StartLocalTime: "", EndLocalTime: "10:00"}})
	require.Error(t, err)

	// UpdateAvailability invalid time format
	_, err = svc.UpdateAvailability(context.Background(), userID, []model.AvailabilitySlot{{Weekday: 1, StartLocalTime: "9am", EndLocalTime: "10:00"}})
	require.Error(t, err)

	// UpdateAvailability end before start
	_, err = svc.UpdateAvailability(context.Background(), userID, []model.AvailabilitySlot{{Weekday: 1, StartLocalTime: "10:00", EndLocalTime: "09:00"}})
	require.Error(t, err)

	// UpdateAvailability invalid timezone
	_, err = svc.UpdateAvailability(context.Background(), userID, []model.AvailabilitySlot{{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "Mars"}})
	require.Error(t, err)

	// UpdateAvailability duplicate
	_, err = svc.UpdateAvailability(context.Background(), userID, []model.AvailabilitySlot{
		{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC"},
		{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC"},
	})
	require.Error(t, err)
}
