package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DiscoveryRepository handles matching/discovery queries.
type DiscoveryRepository struct {
	pool *pgxpool.Pool
}

// NewDiscoveryRepository creates a new DiscoveryRepository.
func NewDiscoveryRepository(pool *pgxpool.Pool) *DiscoveryRepository {
	return &DiscoveryRepository{pool: pool}
}

// Pool returns the underlying connection pool.
func (r *DiscoveryRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// MatchRow represents a single match result from the discovery query.
type MatchRow struct {
	UserID              string
	Handle              string
	CountryCode         *string
	BirthYear           *int
	BirthMonth          *int16
	TotalOverlapMinutes int
}

// OverlapDetailRow represents a single availability overlap slot.
type OverlapDetailRow struct {
	Weekday        int16
	StartUTC       string
	EndUTC         string
	OverlapMinutes int
}

// LanguageRow represents a user's language entry.
type LanguageRow struct {
	LanguageCode string
	Level        int16
	IsNative     bool
	IsTarget     bool
}

const discoverMatchesSQL = `
WITH me_teach AS (
    SELECT ul.language_code
    FROM user_languages ul
    WHERE ul.user_id = $1 AND ul.level >= 4
),
me_target AS (
    SELECT ul.language_code
    FROM user_languages ul
    WHERE ul.user_id = $1 AND ul.is_target = true
),
me_bridge AS (
    SELECT ul.language_code, ul.level
    FROM user_languages ul
    WHERE ul.user_id = $1 AND ul.level >= 3
),
me_slots AS (
    SELECT
        a.weekday,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.start_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS start_utc_min,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.end_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS end_utc_min
    FROM availability_slots a
    WHERE a.user_id = $1
),
candidates AS (
    SELECT DISTINCT p.user_id
    FROM profiles p
    WHERE p.discoverable = true
      AND p.user_id <> $1
      AND NOT EXISTS (
          SELECT 1 FROM user_blocks ub
          WHERE (ub.blocker_id = $1 AND ub.blocked_id = p.user_id)
             OR (ub.blocker_id = p.user_id AND ub.blocked_id = $1)
      )
      AND EXISTS (
          SELECT 1 FROM user_languages ul
          WHERE ul.user_id = p.user_id
            AND SPLIT_PART(ul.language_code, '-', 1) IN (SELECT SPLIT_PART(mt.language_code, '-', 1) FROM me_target mt)
            AND ul.level >= 4
      )
      AND EXISTS (
          SELECT 1 FROM user_languages ul
          WHERE ul.user_id = p.user_id
            AND ul.is_target = true
            AND SPLIT_PART(ul.language_code, '-', 1) IN (SELECT SPLIT_PART(mt.language_code, '-', 1) FROM me_teach mt)
      )
      AND EXISTS (
          SELECT 1 FROM user_languages ul
          JOIN me_bridge mb ON SPLIT_PART(ul.language_code, '-', 1) = SPLIT_PART(mb.language_code, '-', 1)
          WHERE ul.user_id = p.user_id
            AND ul.level >= 3
      )
      AND ($4::text IS NULL OR p.user_id::text > $4)
),
candidate_slots AS (
    SELECT
        c.user_id AS candidate_id,
        a.weekday,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.start_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS start_utc_min,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.end_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS end_utc_min
    FROM candidates c
    JOIN availability_slots a ON a.user_id = c.user_id
),
overlap AS (
    SELECT
        cs.candidate_id,
        ms.weekday,
        GREATEST(0,
            LEAST(ms.end_utc_min, cs.end_utc_min) -
            GREATEST(ms.start_utc_min, cs.start_utc_min)
        ) AS overlap_min
    FROM me_slots ms
    JOIN candidate_slots cs ON ms.weekday = cs.weekday
    WHERE LEAST(ms.end_utc_min, cs.end_utc_min) > GREATEST(ms.start_utc_min, cs.start_utc_min)
),
overlap_totals AS (
    SELECT
        candidate_id,
        SUM(overlap_min)::int AS total_overlap_minutes
    FROM overlap
    GROUP BY candidate_id
    HAVING SUM(overlap_min) >= $2
)
SELECT
    ot.candidate_id::text AS user_id,
    p.handle,
    p.country_code,
    p.birth_year,
    p.birth_month,
    ot.total_overlap_minutes
FROM overlap_totals ot
JOIN profiles p ON p.user_id = ot.candidate_id
ORDER BY ot.total_overlap_minutes DESC, ot.candidate_id ASC
LIMIT $3
`

const overlapDetailsSQL = `
WITH me_slots AS (
    SELECT
        a.weekday,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.start_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS start_utc_min,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.end_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS end_utc_min
    FROM availability_slots a
    WHERE a.user_id = $1
),
candidate_slots AS (
    SELECT
        a.weekday,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.start_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS start_utc_min,
        EXTRACT(EPOCH FROM (
            (CURRENT_DATE + a.weekday * INTERVAL '1 day' + a.end_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS end_utc_min
    FROM availability_slots a
    WHERE a.user_id = $2
)
SELECT
    ms.weekday::smallint AS weekday,
    TO_CHAR(
        INTERVAL '1 minute' * (GREATEST(ms.start_utc_min, cs.start_utc_min) % (24 * 60)),
        'HH24:MI'
    ) AS start_utc,
    TO_CHAR(
        INTERVAL '1 minute' * (LEAST(ms.end_utc_min, cs.end_utc_min) % (24 * 60)),
        'HH24:MI'
    ) AS end_utc,
    GREATEST(0,
        LEAST(ms.end_utc_min, cs.end_utc_min) -
        GREATEST(ms.start_utc_min, cs.start_utc_min)
    )::int AS overlap_minutes
FROM me_slots ms
JOIN candidate_slots cs ON ms.weekday = cs.weekday
WHERE LEAST(ms.end_utc_min, cs.end_utc_min) > GREATEST(ms.start_utc_min, cs.start_utc_min)
ORDER BY ms.weekday, start_utc
`

// DiscoverMatches executes the main matching query.
func (r *DiscoveryRepository) DiscoverMatches(ctx context.Context, userID string, minOverlapMinutes int, limit int, cursor *string) ([]MatchRow, error) {
	var cursorVal *string
	if cursor != nil {
		cursorVal = cursor
	}

	rows, err := r.pool.Query(ctx, discoverMatchesSQL, userID, minOverlapMinutes, limit, cursorVal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MatchRow
	for rows.Next() {
		var m MatchRow
		if err := rows.Scan(&m.UserID, &m.Handle, &m.CountryCode, &m.BirthYear, &m.BirthMonth, &m.TotalOverlapMinutes); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetOverlapDetails returns per-slot overlap details between two users.
func (r *DiscoveryRepository) GetOverlapDetails(ctx context.Context, userID, candidateID string) ([]OverlapDetailRow, error) {
	rows, err := r.pool.Query(ctx, overlapDetailsSQL, userID, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []OverlapDetailRow
	for rows.Next() {
		var o OverlapDetailRow
		if err := rows.Scan(&o.Weekday, &o.StartUTC, &o.EndUTC, &o.OverlapMinutes); err != nil {
			return nil, err
		}
		results = append(results, o)
	}
	return results, rows.Err()
}

// GetUserLanguages returns a user's languages.
func (r *DiscoveryRepository) GetUserLanguages(ctx context.Context, userID string) ([]LanguageRow, error) {
	rows, err := r.pool.Query(ctx, `SELECT language_code, level, is_native, is_target FROM user_languages WHERE user_id = $1 ORDER BY sort_order`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []LanguageRow
	for rows.Next() {
		var l LanguageRow
		if err := rows.Scan(&l.LanguageCode, &l.Level, &l.IsNative, &l.IsTarget); err != nil {
			return nil, err
		}
		results = append(results, l)
	}
	return results, rows.Err()
}

// IsDiscoverable checks if a user's profile exists and is discoverable.
func (r *DiscoveryRepository) IsDiscoverable(ctx context.Context, userID string) (bool, error) {
	var discoverable bool
	err := r.pool.QueryRow(ctx, `SELECT discoverable FROM profiles WHERE user_id = $1`, userID).Scan(&discoverable)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return discoverable, nil
}

// HasTargetLanguages checks if a user has any target languages.
func (r *DiscoveryRepository) HasTargetLanguages(ctx context.Context, userID string) (bool, error) {
	var has bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_languages WHERE user_id = $1 AND is_target = true)`, userID).Scan(&has)
	return has, err
}
