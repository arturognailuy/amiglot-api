package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/model"
)

type ProfileRepository struct {
	pool *pgxpool.Pool
}

func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

func (r *ProfileRepository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *ProfileRepository) LoadUser(ctx context.Context, userID string) (model.User, error) {
	var user model.User
	row := r.pool.QueryRow(ctx, `SELECT id, email FROM users WHERE id = $1`, userID)
	if err := row.Scan(&user.ID, &user.Email); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (r *ProfileRepository) LoadProfile(ctx context.Context, userID string) (model.Profile, error) {
	var profile model.Profile
	row := r.pool.QueryRow(ctx, `
		SELECT handle, birth_year, birth_month, country_code, timezone, discoverable
		FROM profiles
		WHERE user_id = $1
	`, userID)
	if err := row.Scan(&profile.Handle, &profile.BirthYear, &profile.BirthMonth, &profile.CountryCode, &profile.Timezone, &profile.Discoverable); err != nil {
		return model.Profile{}, err
	}
	return profile, nil
}

func (r *ProfileRepository) LoadLanguages(ctx context.Context, userID string) ([]model.Language, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT language_code, level, is_native, is_target, description, sort_order
		FROM user_languages
		WHERE user_id = $1
		ORDER BY sort_order, language_code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := []model.Language{}
	for rows.Next() {
		var lang model.Language
		if err := rows.Scan(&lang.LanguageCode, &lang.Level, &lang.IsNative, &lang.IsTarget, &lang.Description, &lang.SortOrder); err != nil {
			return nil, err
		}
		languages = append(languages, lang)
	}
	return languages, rows.Err()
}

func (r *ProfileRepository) LoadAvailability(ctx context.Context, userID string) ([]model.AvailabilitySlot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT weekday,
		  to_char(start_local_time, 'HH24:MI') as start_local_time,
		  to_char(end_local_time, 'HH24:MI') as end_local_time,
		  timezone,
		  sort_order
		FROM availability_slots
		WHERE user_id = $1
		ORDER BY sort_order, weekday, start_local_time
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	availability := []model.AvailabilitySlot{}
	for rows.Next() {
		var slot model.AvailabilitySlot
		if err := rows.Scan(&slot.Weekday, &slot.StartLocalTime, &slot.EndLocalTime, &slot.Timezone, &slot.SortOrder); err != nil {
			return nil, err
		}
		availability = append(availability, slot)
	}
	return availability, rows.Err()
}

func (r *ProfileRepository) CheckHandleAvailability(ctx context.Context, userID string, handleNorm string) (bool, error) {
	var existingUserID string
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM profiles WHERE handle_norm = $1`, handleNorm).Scan(&existingUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return existingUserID == userID, nil
}

func (r *ProfileRepository) UpsertProfile(ctx context.Context, userID string, profile model.Profile) error {
	_, err := r.pool.Exec(ctx, `
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
	`, userID, profile.Handle, profile.Handle, profile.BirthYear, profile.BirthMonth, profile.CountryCode, profile.Timezone)
	return err
}

func (r *ProfileRepository) ReplaceLanguages(ctx context.Context, userID string, languages []model.Language) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM user_languages WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, lang := range languages {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, description, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, userID, lang.LanguageCode, lang.Level, lang.IsNative, lang.IsTarget, lang.Description, lang.SortOrder)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ProfileRepository) ReplaceAvailability(ctx context.Context, userID string, slots []model.AvailabilitySlot) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM availability_slots WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, slot := range slots {
		_, err := tx.Exec(ctx, `
			INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone, sort_order)
			VALUES ($1, $2, $3::time, $4::time, $5, $6)
		`, userID, slot.Weekday, slot.StartLocalTime, slot.EndLocalTime, slot.Timezone, slot.SortOrder)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ProfileRepository) HasNativeLanguage(ctx context.Context, userID string) (bool, error) {
	var hasNative bool
	row := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM user_languages WHERE user_id = $1 AND is_native = true
		)
	`, userID)
	if err := row.Scan(&hasNative); err != nil {
		return false, err
	}
	return hasNative, nil
}

func (r *ProfileRepository) LoadHandleAndTimezone(ctx context.Context, userID string) (string, string, error) {
	var handle string
	var timezone string
	row := r.pool.QueryRow(ctx, `SELECT handle, timezone FROM profiles WHERE user_id = $1`, userID)
	if err := row.Scan(&handle, &timezone); err != nil {
		return "", "", err
	}
	return handle, timezone, nil
}

func (r *ProfileRepository) UpdateDiscoverable(ctx context.Context, userID string, discoverable bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE profiles SET discoverable = $1, updated_at = now() WHERE user_id = $2`, discoverable, userID)
	return err
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
