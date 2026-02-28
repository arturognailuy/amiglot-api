package http

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gnailuy/amiglot-api/internal/i18n"

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
		return nil, huma.Error503ServiceUnavailable(i18n.T(ctx, "errors.database_unavailable"))
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized(i18n.T(ctx, "errors.missing_user_id"))
	}

	user, err := h.loadUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error401Unauthorized(i18n.T(ctx, "errors.invalid_user_id"))
		}
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_load_user"))
	}

	profile, err := h.loadProfile(ctx, userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_load_profile"))
		}
		profile = profilePayload{
			Handle:       "",
			BirthYear:    nil,
			BirthMonth:   nil,
			CountryCode:  nil,
			Timezone:     "",
			Discoverable: false,
		}
	}

	languages, err := h.loadLanguages(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_load_languages"))
	}

	availability, err := h.loadAvailability(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_load_availability"))
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
		return nil, huma.Error503ServiceUnavailable(i18n.T(ctx, "errors.database_unavailable"))
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized(i18n.T(ctx, "errors.missing_user_id"))
	}

	handle := strings.TrimSpace(input.Handle)
	if handle == "" {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.handle_required"))
	}
	handle = strings.TrimPrefix(handle, "@")
	if len(handle) < handleMinLength || len(handle) > handleMaxLength {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.handle_length"))
	}
	if !handlePattern.MatchString(handle) {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.handle_alphanumeric"))
	}

	handleNorm := strings.ToLower(handle)
	var existingUserID string
	err := h.pool.QueryRow(ctx, `SELECT user_id FROM profiles WHERE handle_norm = $1`, handleNorm).Scan(&existingUserID)
	available := false
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			available = true
		} else {
			return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_check_handle"))
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
		return nil, huma.Error503ServiceUnavailable(i18n.T(ctx, "errors.database_unavailable"))
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized(i18n.T(ctx, "errors.missing_user_id"))
	}

	handle := strings.TrimSpace(input.Body.Handle)
	if handle == "" {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.handle_required"))
	}
	handle = strings.TrimPrefix(handle, "@")
	if len(handle) < handleMinLength || len(handle) > handleMaxLength {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.handle_length"))
	}
	if !handlePattern.MatchString(handle) {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.handle_alphanumeric"))
	}
	handle = strings.ToLower(handle)

	timezone := strings.TrimSpace(input.Body.Timezone)
	if timezone == "" {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.timezone_required"))
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.timezone_invalid"))
	}

	currentYear := time.Now().UTC().Year()
	if input.Body.BirthYear != nil {
		if *input.Body.BirthYear < birthYearMin || *input.Body.BirthYear > currentYear {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.birth_year_range"))
		}
	}
	if input.Body.BirthMonth != nil {
		if *input.Body.BirthMonth < 1 || *input.Body.BirthMonth > 12 {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.birth_month_range"))
		}
	}

	var countryCode *string
	if input.Body.CountryCode != nil {
		trimmed := strings.ToUpper(strings.TrimSpace(*input.Body.CountryCode))
		if trimmed != "" {
			if !countryCodePattern.MatchString(trimmed) {
				return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.country_code_invalid"))
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
			return nil, huma.Error409Conflict(i18n.T(ctx, "errors.handle_taken"))
		}
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_save_profile"))
	}

	if err := h.recalcDiscoverable(ctx, userID); err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_update_discoverable"))
	}

	return h.getProfile(ctx, &profileGetRequest{UserID: userID})
}

func (h *profileHandler) putLanguages(ctx context.Context, input *languagesPutRequest) (*languagesPutResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable(i18n.T(ctx, "errors.database_unavailable"))
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized(i18n.T(ctx, "errors.missing_user_id"))
	}

	languages := input.Body.Languages
	if len(languages) == 0 {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.languages_required"))
	}

	seen := make(map[string]struct{})
	nativeCount := 0
	for _, lang := range languages {
		code := strings.ToLower(strings.TrimSpace(lang.LanguageCode))
		if code == "" {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.language_code_required"))
		}
		if !languageCodePattern.MatchString(code) {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.language_code_invalid"))
		}
		if lang.Level < 0 || lang.Level > 5 {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.level_range"))
		}
		if lang.IsNative && lang.IsTarget {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.language_conflict"))
		}
		if lang.IsNative != (lang.Level == 5) {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.native_level"))
		}
		if lang.IsTarget && lang.Level == 5 {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.native_target"))
		}
		if _, ok := seen[code]; ok {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.language_duplicate"))
		}
		seen[code] = struct{}{}
		if lang.IsNative {
			nativeCount++
		}
	}
	if nativeCount == 0 {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.native_required"))
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_start_tx"))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM user_languages WHERE user_id = $1`, userID); err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_clear_languages"))
	}

	for _, lang := range languages {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, description)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, strings.ToLower(strings.TrimSpace(lang.LanguageCode)), lang.Level, lang.IsNative, lang.IsTarget, lang.Description)
		if err != nil {
			return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_save_languages"))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_save_languages"))
	}

	if err := h.recalcDiscoverable(ctx, userID); err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_update_discoverable"))
	}

	return &languagesPutResponse{
		Body: struct {
			Languages []languagePayload `json:"languages"`
		}{Languages: languages},
	}, nil
}

func (h *profileHandler) putAvailability(ctx context.Context, input *availabilityPutRequest) (*availabilityPutResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable(i18n.T(ctx, "errors.database_unavailable"))
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, huma.Error401Unauthorized(i18n.T(ctx, "errors.missing_user_id"))
	}

	profile, err := h.loadProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.profile_required"))
		}
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_load_profile"))
	}

	slots := input.Body.Availability
	if len(slots) > 14 {
		return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.availability_limit"))
	}
	seen := make(map[string]struct{})
	for i := range slots {
		if slots[i].Weekday < 0 || slots[i].Weekday > 6 {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.weekday_range"))
		}
		start := strings.TrimSpace(slots[i].StartLocalTime)
		end := strings.TrimSpace(slots[i].EndLocalTime)
		if start == "" || end == "" {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.availability_time_required"))
		}
		startTime, err := time.Parse("15:04", start)
		if err != nil {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.start_time_format"))
		}
		endTime, err := time.Parse("15:04", end)
		if err != nil {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.end_time_format"))
		}
		if !startTime.Before(endTime) {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.start_time_order"))
		}

		tz := strings.TrimSpace(slots[i].Timezone)
		if tz == "" {
			tz = profile.Timezone
		}
		if _, err := time.LoadLocation(tz); err != nil {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.timezone_invalid"))
		}
		slots[i].Timezone = tz

		key := fmt.Sprintf("%d|%s|%s|%s", slots[i].Weekday, start, end, tz)
		if _, ok := seen[key]; ok {
			return nil, huma.Error400BadRequest(i18n.T(ctx, "errors.availability_duplicate"))
		}
		seen[key] = struct{}{}
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_start_tx"))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM availability_slots WHERE user_id = $1`, userID); err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_clear_availability"))
	}

	for _, slot := range slots {
		_, err := tx.Exec(ctx, `
			INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone)
			VALUES ($1, $2, $3::time, $4::time, $5)
		`, userID, slot.Weekday, slot.StartLocalTime, slot.EndLocalTime, slot.Timezone)
		if err != nil {
			return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_save_availability"))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, huma.Error500InternalServerError(i18n.T(ctx, "errors.failed_save_availability"))
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
