-- +goose Up
CREATE TABLE IF NOT EXISTS profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  handle TEXT NOT NULL UNIQUE,
  handle_norm TEXT NOT NULL UNIQUE,
  birth_year INT,
  birth_month SMALLINT CHECK (birth_month BETWEEN 1 AND 12),
  country_code CHAR(2),
  timezone TEXT NOT NULL,
  discoverable BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (handle ~ '^[a-zA-Z0-9_]+$')
);

CREATE TABLE IF NOT EXISTS user_languages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  language_code TEXT NOT NULL,
  level SMALLINT NOT NULL CHECK (level BETWEEN 0 AND 5),
  is_native BOOLEAN NOT NULL DEFAULT false,
  is_target BOOLEAN NOT NULL DEFAULT false,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, language_code)
);

CREATE INDEX IF NOT EXISTS user_languages_user_id_idx ON user_languages(user_id);
CREATE INDEX IF NOT EXISTS user_languages_language_idx ON user_languages(language_code, level);

CREATE TABLE IF NOT EXISTS availability_slots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
  start_local_time TIME NOT NULL,
  end_local_time TIME NOT NULL,
  timezone TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS availability_user_idx ON availability_slots(user_id);
CREATE INDEX IF NOT EXISTS availability_local_idx ON availability_slots(weekday, start_local_time, end_local_time);

-- +goose Down
DROP TABLE IF EXISTS availability_slots;
DROP TABLE IF EXISTS user_languages;
DROP TABLE IF EXISTS profiles;
