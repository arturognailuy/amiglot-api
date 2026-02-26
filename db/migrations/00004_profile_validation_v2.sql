-- +goose Up
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'profiles_handle_alnum_check'
  ) THEN
    ALTER TABLE profiles
      ADD CONSTRAINT profiles_handle_alnum_check CHECK (handle ~ '^[a-zA-Z0-9]+$');
  END IF;
END $$;

-- +goose Down
ALTER TABLE profiles
  DROP CONSTRAINT IF EXISTS profiles_handle_alnum_check;
