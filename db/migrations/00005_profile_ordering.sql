-- +goose Up
ALTER TABLE user_languages ADD COLUMN IF NOT EXISTS sort_order INT;
ALTER TABLE availability_slots ADD COLUMN IF NOT EXISTS sort_order INT;

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at, language_code) AS rn
  FROM user_languages
)
UPDATE user_languages ul
SET sort_order = ranked.rn
FROM ranked
WHERE ul.id = ranked.id
  AND (ul.sort_order IS NULL OR ul.sort_order = 0);

WITH grouped AS (
  SELECT user_id, start_local_time, end_local_time, timezone, MIN(created_at) AS min_created_at
  FROM availability_slots
  GROUP BY user_id, start_local_time, end_local_time, timezone
), ranked AS (
  SELECT user_id,
         start_local_time,
         end_local_time,
         timezone,
         DENSE_RANK() OVER (
           PARTITION BY user_id
           ORDER BY min_created_at, start_local_time, end_local_time, timezone
         ) AS rn
  FROM grouped
)
UPDATE availability_slots a
SET sort_order = ranked.rn
FROM ranked
WHERE a.user_id = ranked.user_id
  AND a.start_local_time = ranked.start_local_time
  AND a.end_local_time = ranked.end_local_time
  AND a.timezone = ranked.timezone
  AND (a.sort_order IS NULL OR a.sort_order = 0);

ALTER TABLE user_languages ALTER COLUMN sort_order SET NOT NULL;
ALTER TABLE availability_slots ALTER COLUMN sort_order SET NOT NULL;

-- +goose Down
ALTER TABLE availability_slots DROP COLUMN IF EXISTS sort_order;
ALTER TABLE user_languages DROP COLUMN IF EXISTS sort_order;
