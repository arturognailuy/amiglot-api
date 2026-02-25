-- +goose Up
ALTER TABLE profiles
  ADD CONSTRAINT profiles_handle_alnum_check CHECK (handle ~ '^[a-zA-Z0-9]+$');

-- +goose Down
ALTER TABLE profiles
  DROP CONSTRAINT IF EXISTS profiles_handle_alnum_check;
