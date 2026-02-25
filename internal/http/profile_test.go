package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileValidation(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	h := &profileHandler{pool: pool}

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "handle is required")

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "bad-handle", Timezone: "America/Vancouver"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "handle must be alphanumeric")

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "arturo"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timezone is required")
}

func TestLanguagesValidation(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user2@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	h := &profileHandler{pool: pool}

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "languages are required")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "en", Level: 1, IsNative: false}, {LanguageCode: "en", Level: 1, IsNative: true}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate language_code")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "en", Level: 1, IsNative: false}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "native language")
}

func TestAvailabilityRequiresProfile(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user3@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	h := &profileHandler{pool: pool}

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 1, StartLocalTime: "18:00", EndLocalTime: "20:00", Timezone: ""}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile is required")
}
