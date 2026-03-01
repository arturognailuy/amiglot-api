package http

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

func newProfileHandler(pool *pgxpool.Pool) *profileHandler {
	repo := repository.NewProfileRepository(pool)
	svc := service.NewProfileService(repo)
	return &profileHandler{svc: svc}
}

func TestProfileValidation(t *testing.T) {
	pool := openTestPool(t)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

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

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user2@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "languages are required")

	_, err = h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: []languagePayload{{LanguageCode: "en", Level: 5, IsNative: true, IsTarget: false}, {LanguageCode: "en", Level: 5, IsNative: true, IsTarget: false}}}})
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

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user3@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 1, StartLocalTime: "18:00", EndLocalTime: "20:00", Timezone: ""}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "profile is required")
}

func TestGetProfile_NotFound(t *testing.T) {
	pool := openTestPool(t)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ($1) RETURNING id`, "user4@example.com").Scan(&userID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

	resp, err := h.getProfile(context.Background(), &profileGetRequest{UserID: userID})
	require.NoError(t, err)
	require.Equal(t, userID, resp.Body.User.ID)
	require.Equal(t, "user4@example.com", resp.Body.User.Email)
	require.Equal(t, "", resp.Body.Profile.Handle)
	require.Equal(t, "", resp.Body.Profile.Timezone)
	require.False(t, resp.Body.Profile.Discoverable)
	require.Len(t, resp.Body.Languages, 0)
	require.Len(t, resp.Body.Availability, 0)
}

func TestProfileFlow_Success(t *testing.T) {
	pool := openTestPool(t)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ($1) RETURNING id`, "user5@example.com").Scan(&userID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

	birthYear := 1992
	birthMonth := int16(6)
	country := "CA"
	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "@Arturo", BirthYear: &birthYear, BirthMonth: &birthMonth, CountryCode: &country, Timezone: "America/Vancouver"}})
	require.NoError(t, err)

	languages := []languagePayload{{LanguageCode: "en", Level: 5, IsNative: true, IsTarget: false}}
	langResp, err := h.putLanguages(context.Background(), &languagesPutRequest{UserID: userID, Body: struct {
		Languages []languagePayload `json:"languages"`
	}{Languages: languages}})
	require.NoError(t, err)
	require.Len(t, langResp.Body.Languages, 1)

	availability := []availabilityPayload{{Weekday: 2, StartLocalTime: "09:00", EndLocalTime: "12:00", Timezone: ""}}
	availResp, err := h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: availability}})
	require.NoError(t, err)
	require.Len(t, availResp.Body.Availability, 1)
	require.Equal(t, "America/Vancouver", availResp.Body.Availability[0].Timezone)

	resp, err := h.getProfile(context.Background(), &profileGetRequest{UserID: userID})
	require.NoError(t, err)
	require.Equal(t, userID, resp.Body.User.ID)
	require.Equal(t, "user5@example.com", resp.Body.User.Email)
	require.Equal(t, "arturo", resp.Body.Profile.Handle)
	require.Equal(t, "America/Vancouver", resp.Body.Profile.Timezone)
	require.True(t, resp.Body.Profile.Discoverable)
	require.Len(t, resp.Body.Languages, 1)
	require.Len(t, resp.Body.Availability, 1)
}

func TestAvailabilityValidation(t *testing.T) {
	pool := openTestPool(t)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ($1) RETURNING id`, "user6@example.com").Scan(&userID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "tester", Timezone: "UTC"}})
	require.NoError(t, err)

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 7, StartLocalTime: "09:00", EndLocalTime: "10:00", Timezone: "UTC"}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "weekday must be between 0 and 6")

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 1, StartLocalTime: "", EndLocalTime: "10:00", Timezone: "UTC"}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "start_local_time and end_local_time are required")

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 1, StartLocalTime: "9am", EndLocalTime: "10:00", Timezone: "UTC"}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "start_local_time must be HH:MM")

	_, err = h.putAvailability(context.Background(), &availabilityPutRequest{UserID: userID, Body: struct {
		Availability []availabilityPayload `json:"availability"`
	}{Availability: []availabilityPayload{{Weekday: 1, StartLocalTime: "12:00", EndLocalTime: "10:00", Timezone: "UTC"}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "start_local_time must be before end_local_time")
}

func TestIsUniqueViolation(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"}
	require.True(t, repository.IsUniqueViolation(err))
	require.False(t, repository.IsUniqueViolation(nil))
}

func TestHandleAvailability(t *testing.T) {
	pool := openTestPool(t)

	var userID string
	err := pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user7@example.com') RETURNING id`).Scan(&userID)
	require.NoError(t, err)

	var otherUserID string
	err = pool.QueryRow(context.Background(), `INSERT INTO users (email) VALUES ('user8@example.com') RETURNING id`).Scan(&otherUserID)
	require.NoError(t, err)

	h := newProfileHandler(pool)

	_, err = h.putProfile(context.Background(), &profileUpdateRequest{UserID: userID, Body: struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}{Handle: "Arturo", Timezone: "UTC"}})
	require.NoError(t, err)

	resp, err := h.checkHandleAvailability(context.Background(), &handleCheckRequest{UserID: userID, Handle: "arturo"})
	require.NoError(t, err)
	require.True(t, resp.Body.Available)

	resp, err = h.checkHandleAvailability(context.Background(), &handleCheckRequest{UserID: otherUserID, Handle: "arturo"})
	require.NoError(t, err)
	require.False(t, resp.Body.Available)

	resp, err = h.checkHandleAvailability(context.Background(), &handleCheckRequest{UserID: otherUserID, Handle: "fresh"})
	require.NoError(t, err)
	require.True(t, resp.Body.Available)
}
