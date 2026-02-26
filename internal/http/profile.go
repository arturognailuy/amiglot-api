package http

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	handleMinLength = 3
	handleMaxLength = 20
	birthYearMin    = 1900
)

var (
	handlePattern       = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	languageCodePattern = regexp.MustCompile(`^[a-z]{2,3}$`)
	countryCodePattern  = regexp.MustCompile(`^[A-Z]{2}$`)
)

type profileHandler struct {
	pool *pgxpool.Pool
}

type profilePayload struct {
	Handle       string  `json:"handle"`
	BirthYear    *int    `json:"birth_year,omitempty"`
	BirthMonth   *int16  `json:"birth_month,omitempty"`
	CountryCode  *string `json:"country_code,omitempty"`
	Timezone     string  `json:"timezone"`
	Discoverable bool    `json:"discoverable"`
}

type languagePayload struct {
	LanguageCode string  `json:"language_code"`
	Level        int16   `json:"level"`
	IsNative     bool    `json:"is_native"`
	IsTarget     bool    `json:"is_target"`
	Description  *string `json:"description,omitempty"`
}

type availabilityPayload struct {
	Weekday        int16  `json:"weekday"`
	StartLocalTime string `json:"start_local_time"`
	EndLocalTime   string `json:"end_local_time"`
	Timezone       string `json:"timezone"`
}

type userPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type profileResponse struct {
	Body struct {
		User         userPayload           `json:"user"`
		Profile      profilePayload        `json:"profile"`
		Languages    []languagePayload     `json:"languages"`
		Availability []availabilityPayload `json:"availability"`
	}
}

type languagesPutResponse struct {
	Body struct {
		Languages []languagePayload `json:"languages"`
	}
}

type availabilityPutResponse struct {
	Body struct {
		Availability []availabilityPayload `json:"availability"`
	}
}

type profileUpdateRequest struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}
}

type profileGetRequest struct {
	UserID string `header:"X-User-Id"`
}

type languagesPutRequest struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		Languages []languagePayload `json:"languages"`
	}
}

type availabilityPutRequest struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		Availability []availabilityPayload `json:"availability"`
	}
}

type handleCheckRequest struct {
	UserID string `header:"X-User-Id"`
	Handle string `query:"handle"`
}

type handleCheckResponse struct {
	Body struct {
		Available bool `json:"available"`
	}
}

func registerProfileRoutes(api huma.API, pool *pgxpool.Pool) {
	h := &profileHandler{pool: pool}

	huma.Get(api, "/profile", h.getProfile)
	huma.Get(api, "/profile/handle/check", h.checkHandleAvailability)
	huma.Put(api, "/profile", h.putProfile)
	huma.Put(api, "/profile/languages", h.putLanguages)
	huma.Put(api, "/profile/availability", h.putAvailability)
}

func (h *profileHandler) getProfile(ctx context.Context, input *profileGetRequest) (*profileResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized("missing user id")
	}

	user, err := h.loadUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error401Unauthorized("invalid user id")
		}
		return nil, huma.Error500InternalServerError("failed to load user")
	}

	profile, err := h.loadProfile(ctx, userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error500InternalServerError("failed to load profile")
		}
		profile = profilePayload{
			Handle:       "",
			BirthYear:    nil,
			BirthMonth:   nil,
			CountryCode:  nil,
			Timezone:     "America/Vancouver",
			Discoverable: false,
		}
	}

	languages, err := h.loadLanguages(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load languages")
	}

	availability, err := h.loadAvailability(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load availability")
	}

	return &profileResponse{
		Body: struct {
			User         userPayload           `json:"user"`
			Profile      profilePayload        `json:"profile"`
			Languages    []languagePayload     `json:"languages"`
			Availability []availabilityPayload `json:"availability"`
		}{
			User:         user,
			Profile:      profile,
			Languages:    languages,
			Availability: availability,
		},
	}, nil
}

func (h *profileHandler) checkHandleAvailability(ctx context.Context, input *handleCheckRequest) (*handleCheckResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized("missing user id")
	}

	handle := strings.TrimSpace(input.Handle)
	if handle == "" {
		return nil, huma.Error400BadRequest("handle is required")
	}
	handle = strings.TrimPrefix(handle, "@")
	if len(handle) < handleMinLength || len(handle) > handleMaxLength {
		return nil, huma.Error400BadRequest("handle must be 3-20 characters")
	}
	if !handlePattern.MatchString(handle) {
		return nil, huma.Error400BadRequest("handle must be alphanumeric")
	}

	handleNorm := strings.ToLower(handle)
	var existingUserID string
	err := h.pool.QueryRow(ctx, `SELECT user_id FROM profiles WHERE handle_norm = $1`, handleNorm).Scan(&existingUserID)
	available := false
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			available = true
		} else {
			return nil, huma.Error500InternalServerError("failed to check handle")
		}
	} else {
		available = existingUserID == userID
	}

	return &handleCheckResponse{
		Body: struct {
			Available bool `json:"available"`
		}{Available: available},
	}, nil
}

func (h *profileHandler) putProfile(ctx context.Context, input *profileUpdateRequest) (*profileResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized("missing user id")
	}

	handle := strings.TrimSpace(input.Body.Handle)
	if handle == "" {
		return nil, huma.Error400BadRequest("handle is required")
	}
	handle = strings.TrimPrefix(handle, "@")
	if len(handle) < handleMinLength || len(handle) > handleMaxLength {
		return nil, huma.Error400BadRequest("handle must be 3-20 characters")
	}
	if !handlePattern.MatchString(handle) {
		return nil, huma.Error400BadRequest("handle must be alphanumeric")
	}
	handle = strings.ToLower(handle)

	timezone := strings.TrimSpace(input.Body.Timezone)
	if timezone == "" {
		return nil, huma.Error400BadRequest("timezone is required")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, huma.Error400BadRequest("timezone is invalid")
	}

	currentYear := time.Now().UTC().Year()
	if input.Body.BirthYear != nil {
		if *input.Body.BirthYear < birthYearMin || *input.Body.BirthYear > currentYear {
			return nil, huma.Error400BadRequest("birth_year must be between 1900 and current year")
		}
	}
	if input.Body.BirthMonth != nil {
		if *input.Body.BirthMonth < 1 || *input.Body.BirthMonth > 12 {
			return nil, huma.Error400BadRequest("birth_month must be between 1 and 12")
		}
	}

	var countryCode *string
	if input.Body.CountryCode != nil {
		trimmed := strings.ToUpper(strings.TrimSpace(*input.Body.CountryCode))
		if trimmed != "" {
			if !countryCodePattern.MatchString(trimmed) {
				return nil, huma.Error400BadRequest("country_code must be ISO-3166 alpha-2")
			}
			countryCode = &trimmed
		}
	}

	handleNorm := handle

	_, err := h.pool.Exec(ctx, `
		INSERT INTO profiles (user_id, handle, handle_norm, birth_year, birth_month, country_code, timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
		  handle = EXCLUDED.handle,
		  handle_norm = EXCLUDED.handle_norm,
		  birth_year = EXCLUDED.birth_year,
		  birth_month = EXCLUDED.birth_month,
		  country_code = EXCLUDED.country_code,
		  timezone = EXCLUDED.timezone,
		  updated_at = now()
	`, userID, handle, handleNorm, input.Body.BirthYear, input.Body.BirthMonth, countryCode, timezone)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, huma.Error409Conflict("handle is already taken")
		}
		return nil, huma.Error500InternalServerError("failed to save profile")
	}

	if err := h.recalcDiscoverable(ctx, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to update discoverable status")
	}

	return h.getProfile(ctx, &profileGetRequest{UserID: userID})
}

func (h *profileHandler) putLanguages(ctx context.Context, input *languagesPutRequest) (*languagesPutResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized("missing user id")
	}

	languages := input.Body.Languages
	if len(languages) == 0 {
		return nil, huma.Error400BadRequest("languages are required")
	}

	seen := make(map[string]struct{})
	nativeCount := 0
	for _, lang := range languages {
		code := strings.ToLower(strings.TrimSpace(lang.LanguageCode))
		if code == "" {
			return nil, huma.Error400BadRequest("language_code is required")
		}
		if !languageCodePattern.MatchString(code) {
			return nil, huma.Error400BadRequest("language_code must be ISO-639 (2-3 letters)")
		}
		if lang.Level < 0 || lang.Level > 5 {
			return nil, huma.Error400BadRequest("level must be between 0 and 5")
		}
		if lang.IsNative && lang.IsTarget {
			return nil, huma.Error400BadRequest("language cannot be both native and target")
		}
		if lang.IsNative != (lang.Level == 5) {
			return nil, huma.Error400BadRequest("native level must be level 5")
		}
		if lang.IsTarget && lang.Level == 5 {
			return nil, huma.Error400BadRequest("native language cannot be target")
		}
		if _, ok := seen[code]; ok {
			return nil, huma.Error400BadRequest("duplicate language_code")
		}
		seen[code] = struct{}{}
		if lang.IsNative {
			nativeCount++
		}
	}
	if nativeCount == 0 {
		return nil, huma.Error400BadRequest("at least one native language is required")
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM user_languages WHERE user_id = $1`, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear languages")
	}

	for _, lang := range languages {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, description)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, strings.ToLower(strings.TrimSpace(lang.LanguageCode)), lang.Level, lang.IsNative, lang.IsTarget, lang.Description)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to save languages")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, huma.Error500InternalServerError("failed to save languages")
	}

	if err := h.recalcDiscoverable(ctx, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to update discoverable status")
	}

	return &languagesPutResponse{
		Body: struct {
			Languages []languagePayload `json:"languages"`
		}{Languages: languages},
	}, nil
}

func (h *profileHandler) putAvailability(ctx context.Context, input *availabilityPutRequest) (*availabilityPutResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized("missing user id")
	}

	profile, err := h.loadProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error400BadRequest("profile is required before availability")
		}
		return nil, huma.Error500InternalServerError("failed to load profile")
	}

	slots := input.Body.Availability
	if len(slots) > 14 {
		return nil, huma.Error400BadRequest("availability is limited to 14 slots")
	}
	seen := make(map[string]struct{})
	for i := range slots {
		if slots[i].Weekday < 0 || slots[i].Weekday > 6 {
			return nil, huma.Error400BadRequest("weekday must be between 0 and 6")
		}
		start := strings.TrimSpace(slots[i].StartLocalTime)
		end := strings.TrimSpace(slots[i].EndLocalTime)
		if start == "" || end == "" {
			return nil, huma.Error400BadRequest("start_local_time and end_local_time are required")
		}
		startTime, err := time.Parse("15:04", start)
		if err != nil {
			return nil, huma.Error400BadRequest("start_local_time must be HH:MM")
		}
		endTime, err := time.Parse("15:04", end)
		if err != nil {
			return nil, huma.Error400BadRequest("end_local_time must be HH:MM")
		}
		if !startTime.Before(endTime) {
			return nil, huma.Error400BadRequest("start_local_time must be before end_local_time")
		}

		tz := strings.TrimSpace(slots[i].Timezone)
		if tz == "" {
			tz = profile.Timezone
		}
		if _, err := time.LoadLocation(tz); err != nil {
			return nil, huma.Error400BadRequest("timezone is invalid")
		}
		slots[i].Timezone = tz

		key := fmt.Sprintf("%d|%s|%s|%s", slots[i].Weekday, start, end, tz)
		if _, ok := seen[key]; ok {
			return nil, huma.Error400BadRequest("availability slot is duplicate")
		}
		seen[key] = struct{}{}
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM availability_slots WHERE user_id = $1`, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear availability")
	}

	for _, slot := range slots {
		_, err := tx.Exec(ctx, `
			INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone)
			VALUES ($1, $2, $3::time, $4::time, $5)
		`, userID, slot.Weekday, slot.StartLocalTime, slot.EndLocalTime, slot.Timezone)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to save availability")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, huma.Error500InternalServerError("failed to save availability")
	}

	return &availabilityPutResponse{
		Body: struct {
			Availability []availabilityPayload `json:"availability"`
		}{Availability: slots},
	}, nil
}

func (h *profileHandler) loadUser(ctx context.Context, userID string) (userPayload, error) {
	var user userPayload
	row := h.pool.QueryRow(ctx, `SELECT id, email FROM users WHERE id = $1`, userID)
	if err := row.Scan(&user.ID, &user.Email); err != nil {
		return userPayload{}, err
	}
	return user, nil
}

func (h *profileHandler) loadProfile(ctx context.Context, userID string) (profilePayload, error) {
	var profile profilePayload
	row := h.pool.QueryRow(ctx, `
		SELECT handle, birth_year, birth_month, country_code, timezone, discoverable
		FROM profiles
		WHERE user_id = $1
	`, userID)
	if err := row.Scan(&profile.Handle, &profile.BirthYear, &profile.BirthMonth, &profile.CountryCode, &profile.Timezone, &profile.Discoverable); err != nil {
		return profilePayload{}, err
	}
	return profile, nil
}

func (h *profileHandler) loadLanguages(ctx context.Context, userID string) ([]languagePayload, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT language_code, level, is_native, is_target, description
		FROM user_languages
		WHERE user_id = $1
		ORDER BY language_code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := []languagePayload{}
	for rows.Next() {
		var lang languagePayload
		if err := rows.Scan(&lang.LanguageCode, &lang.Level, &lang.IsNative, &lang.IsTarget, &lang.Description); err != nil {
			return nil, err
		}
		languages = append(languages, lang)
	}
	return languages, rows.Err()
}

func (h *profileHandler) loadAvailability(ctx context.Context, userID string) ([]availabilityPayload, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT weekday,
		  to_char(start_local_time, 'HH24:MI') as start_local_time,
		  to_char(end_local_time, 'HH24:MI') as end_local_time,
		  timezone
		FROM availability_slots
		WHERE user_id = $1
		ORDER BY weekday, start_local_time
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	availability := []availabilityPayload{}
	for rows.Next() {
		var slot availabilityPayload
		if err := rows.Scan(&slot.Weekday, &slot.StartLocalTime, &slot.EndLocalTime, &slot.Timezone); err != nil {
			return nil, err
		}
		availability = append(availability, slot)
	}
	return availability, rows.Err()
}

func (h *profileHandler) recalcDiscoverable(ctx context.Context, userID string) error {
	var hasNative bool
	row := h.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM user_languages WHERE user_id = $1 AND is_native = true
		)
	`, userID)
	if err := row.Scan(&hasNative); err != nil {
		return err
	}

	var handle string
	var timezone string
	row = h.pool.QueryRow(ctx, `SELECT handle, timezone FROM profiles WHERE user_id = $1`, userID)
	if err := row.Scan(&handle, &timezone); err != nil {
		return err
	}

	discoverable := hasNative && strings.TrimSpace(handle) != "" && strings.TrimSpace(timezone) != ""
	_, err := h.pool.Exec(ctx, `UPDATE profiles SET discoverable = $1, updated_at = now() WHERE user_id = $2`, discoverable, userID)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
