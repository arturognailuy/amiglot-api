package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gnailuy/amiglot-api/internal/model"
	"github.com/gnailuy/amiglot-api/internal/repository"
)

const (
	handleMinLength = 3
	handleMaxLength = 20
	birthYearMin    = 1900
)

var (
	handlePattern       = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	languageCodePattern = regexp.MustCompile(`^[a-z]{2,3}([_-][a-z0-9]{2,8})*$`)
	countryCodePattern  = regexp.MustCompile(`^[A-Z]{2}$`)
)

type ProfileService struct {
	repo *repository.ProfileRepository
}

func NewProfileService(repo *repository.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (model.User, model.Profile, []model.Language, []model.AvailabilitySlot, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}

	user, err := s.repo.LoadUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.Profile{}, nil, nil, &Error{Status: 401, Key: "errors.invalid_user_id"}
		}
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 500, Key: "errors.failed_load_user", Err: err}
	}

	profile, err := s.repo.LoadProfile(ctx, userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, model.Profile{}, nil, nil, &Error{Status: 500, Key: "errors.failed_load_profile", Err: err}
		}
		profile = model.Profile{
			Handle:       "",
			BirthYear:    nil,
			BirthMonth:   nil,
			CountryCode:  nil,
			Timezone:     "",
			Discoverable: false,
		}
	}

	languages, err := s.repo.LoadLanguages(ctx, userID)
	if err != nil {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 500, Key: "errors.failed_load_languages", Err: err}
	}

	availability, err := s.repo.LoadAvailability(ctx, userID)
	if err != nil {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 500, Key: "errors.failed_load_availability", Err: err}
	}

	return user, profile, languages, availability, nil
}

func (s *ProfileService) CheckHandleAvailability(ctx context.Context, userID string, handle string) (bool, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return false, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return false, &Error{Status: 401, Key: "errors.missing_user_id"}
	}

	handle = strings.TrimSpace(handle)
	if handle == "" {
		return false, &Error{Status: 400, Key: "errors.handle_required"}
	}
	handle = strings.TrimPrefix(handle, "@")
	if len(handle) < handleMinLength || len(handle) > handleMaxLength {
		return false, &Error{Status: 400, Key: "errors.handle_length"}
	}
	if !handlePattern.MatchString(handle) {
		return false, &Error{Status: 400, Key: "errors.handle_alphanumeric"}
	}

	handleNorm := strings.ToLower(handle)
	available, err := s.repo.CheckHandleAvailability(ctx, userID, handleNorm)
	if err != nil {
		return false, &Error{Status: 500, Key: "errors.failed_check_handle", Err: err}
	}

	return available, nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID string, profile model.Profile) (model.User, model.Profile, []model.Language, []model.AvailabilitySlot, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}

	handle := strings.TrimSpace(profile.Handle)
	if handle == "" {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.handle_required"}
	}
	handle = strings.TrimPrefix(handle, "@")
	if len(handle) < handleMinLength || len(handle) > handleMaxLength {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.handle_length"}
	}
	if !handlePattern.MatchString(handle) {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.handle_alphanumeric"}
	}
	handle = strings.ToLower(handle)

	timezone := strings.TrimSpace(profile.Timezone)
	if timezone == "" {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.timezone_required"}
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.timezone_invalid"}
	}

	currentYear := time.Now().UTC().Year()
	if profile.BirthYear != nil {
		if *profile.BirthYear < birthYearMin || *profile.BirthYear > currentYear {
			return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.birth_year_range"}
		}
	}
	if profile.BirthMonth != nil {
		if *profile.BirthMonth < 1 || *profile.BirthMonth > 12 {
			return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.birth_month_range"}
		}
	}

	var countryCode *string
	if profile.CountryCode != nil {
		trimmed := strings.ToUpper(strings.TrimSpace(*profile.CountryCode))
		if trimmed != "" {
			if !countryCodePattern.MatchString(trimmed) {
				return model.User{}, model.Profile{}, nil, nil, &Error{Status: 400, Key: "errors.country_code_invalid"}
			}
			countryCode = &trimmed
		}
	}

	profile.Handle = handle
	profile.Timezone = timezone
	profile.CountryCode = countryCode

	if err := s.repo.UpsertProfile(ctx, userID, profile); err != nil {
		if repository.IsUniqueViolation(err) {
			return model.User{}, model.Profile{}, nil, nil, &Error{Status: 409, Key: "errors.handle_taken"}
		}
		return model.User{}, model.Profile{}, nil, nil, &Error{Status: 500, Key: "errors.failed_save_profile", Err: err}
	}

	if err := s.recalcDiscoverable(ctx, userID); err != nil {
		return model.User{}, model.Profile{}, nil, nil, err
	}

	return s.GetProfile(ctx, userID)
}

func (s *ProfileService) UpdateLanguages(ctx context.Context, userID string, languages []model.Language) ([]model.Language, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return nil, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}

	if len(languages) == 0 {
		return nil, &Error{Status: 400, Key: "errors.languages_required"}
	}

	normalizedLanguages := make([]model.Language, 0, len(languages))
	seen := make(map[string]struct{})
	nativeCount := 0
	for _, lang := range languages {
		code := normalizeLanguageCode(lang.LanguageCode)
		if code == "" {
			return nil, &Error{Status: 400, Key: "errors.language_code_required"}
		}
		if !languageCodePattern.MatchString(code) {
			return nil, &Error{Status: 400, Key: "errors.language_code_invalid"}
		}
		if lang.Level < 0 || lang.Level > 5 {
			return nil, &Error{Status: 400, Key: "errors.level_range"}
		}
		if lang.IsNative && lang.IsTarget {
			return nil, &Error{Status: 400, Key: "errors.language_conflict"}
		}
		if lang.IsNative != (lang.Level == 5) {
			return nil, &Error{Status: 400, Key: "errors.native_level"}
		}
		if lang.IsTarget && lang.Level == 5 {
			return nil, &Error{Status: 400, Key: "errors.native_target"}
		}
		if _, ok := seen[code]; ok {
			return nil, &Error{Status: 400, Key: "errors.language_duplicate"}
		}
		seen[code] = struct{}{}
		if lang.IsNative {
			nativeCount++
		}
		normalizedLang := lang
		normalizedLang.LanguageCode = code
		normalizedLanguages = append(normalizedLanguages, normalizedLang)
	}
	if nativeCount == 0 {
		return nil, &Error{Status: 400, Key: "errors.native_required"}
	}

	if err := s.repo.ReplaceLanguages(ctx, userID, normalizedLanguages); err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_save_languages", Err: err}
	}

	if err := s.recalcDiscoverable(ctx, userID); err != nil {
		return nil, err
	}

	return normalizedLanguages, nil
}

func (s *ProfileService) UpdateAvailability(ctx context.Context, userID string, slots []model.AvailabilitySlot) ([]model.AvailabilitySlot, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return nil, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "undefined" || userID == "null" {
		return nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}

	profile, err := s.repo.LoadProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &Error{Status: 400, Key: "errors.profile_required"}
		}
		return nil, &Error{Status: 500, Key: "errors.failed_load_profile", Err: err}
	}

	if len(slots) > 14 {
		return nil, &Error{Status: 400, Key: "errors.availability_limit"}
	}
	seen := make(map[string]struct{})
	for i := range slots {
		if slots[i].Weekday < 0 || slots[i].Weekday > 6 {
			return nil, &Error{Status: 400, Key: "errors.weekday_range"}
		}
		start := strings.TrimSpace(slots[i].StartLocalTime)
		end := strings.TrimSpace(slots[i].EndLocalTime)
		if start == "" || end == "" {
			return nil, &Error{Status: 400, Key: "errors.availability_time_required"}
		}
		startTime, err := time.Parse("15:04", start)
		if err != nil {
			return nil, &Error{Status: 400, Key: "errors.start_time_format"}
		}
		endTime, err := time.Parse("15:04", end)
		if err != nil {
			return nil, &Error{Status: 400, Key: "errors.end_time_format"}
		}
		if !startTime.Before(endTime) {
			return nil, &Error{Status: 400, Key: "errors.start_time_order"}
		}

		tz := strings.TrimSpace(slots[i].Timezone)
		if tz == "" {
			tz = profile.Timezone
		}
		if _, err := time.LoadLocation(tz); err != nil {
			return nil, &Error{Status: 400, Key: "errors.timezone_invalid"}
		}
		slots[i].Timezone = tz

		key := fmt.Sprintf("%d|%s|%s|%s", slots[i].Weekday, start, end, tz)
		if _, ok := seen[key]; ok {
			return nil, &Error{Status: 400, Key: "errors.availability_duplicate"}
		}
		seen[key] = struct{}{}
	}

	if err := s.repo.ReplaceAvailability(ctx, userID, slots); err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_save_availability", Err: err}
	}

	return slots, nil
}

func (s *ProfileService) recalcDiscoverable(ctx context.Context, userID string) error {
	hasNative, err := s.repo.HasNativeLanguage(ctx, userID)
	if err != nil {
		return &Error{Status: 500, Key: "errors.failed_update_discoverable", Err: err}
	}

	handle, timezone, err := s.repo.LoadHandleAndTimezone(ctx, userID)
	if err != nil {
		return &Error{Status: 500, Key: "errors.failed_update_discoverable", Err: err}
	}

	discoverable := hasNative && strings.TrimSpace(handle) != "" && strings.TrimSpace(timezone) != ""
	if err := s.repo.UpdateDiscoverable(ctx, userID, discoverable); err != nil {
		return &Error{Status: 500, Key: "errors.failed_update_discoverable", Err: err}
	}
	return nil
}

func normalizeLanguageCode(code string) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	return strings.ReplaceAll(normalized, "_", "-")
}
