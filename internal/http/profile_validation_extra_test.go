package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileValidation_AdditionalErrors(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ($1) RETURNING id`, "extra@example.com").Scan(&userID)
	require.NoError(t, err)

	h := &profileHandler{pool: pool}

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "ab", Timezone: "UTC"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "handle must be 3-20 characters")

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "validhandle", Timezone: "Mars/Phobos"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timezone is invalid")

	year := 1800
	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "validhandle", BirthYear: &year, Timezone: "UTC"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "birth_year must be between 1900 and current year")

	month := int16(13)
	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "validhandle", BirthMonth: &month, Timezone: "UTC"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "birth_month must be between 1 and 12")

	badCountry := "USA"
	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "validhandle", CountryCode: &badCountry, Timezone: "UTC"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "country_code must be ISO-3166 alpha-2")
}

func TestLanguagesValidation_AdditionalErrors(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ($1) RETURNING id`, "extra2@example.com").Scan(&userID)
	require.NoError(t, err)

	h := &profileHandler{pool: pool}

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "pt-br", Level: 5, IsNative: true}}}})
	require.NoError(t, err)

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "english", Level: 2, IsNative: false}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "language_code must be BCP-47")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "en", Level: 6, IsNative: false}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "level must be between 0 and 5")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "en", Level: 5, IsNative: true, IsTarget: true}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "language cannot be both native and target")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "en", Level: 4, IsNative: true}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "native level must be level 5")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "", Level: 1, IsNative: true}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "language_code is required")
}

func TestAvailabilityValidation_AdditionalErrors(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ($1) RETURNING id`, "extra3@example.com").Scan(&userID)
	require.NoError(t, err)

	h := &profileHandler{pool: pool}

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "validhandle", Timezone: "UTC"}})
	require.NoError(t, err)

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "Mars/Phobos"}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timezone is invalid")

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{
		{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC"},
		{Weekday: 1, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC"},
	}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "availability slot is duplicate")
}

func TestHandleAvailability_MissingUserID(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	h := &profileHandler{pool: pool}

	_, err := h.checkHandleAvailability(context.Background(), &handleCheckRequest{UserID: "", Handle: "valid"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing user id")
}
