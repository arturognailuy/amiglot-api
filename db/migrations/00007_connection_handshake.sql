-- +goose Up

-- Tables for match requests, matches, and messages
CREATE TABLE IF NOT EXISTS match_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('pending','accepted','declined','canceled')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  responded_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS match_requests_unique_pending
  ON match_requests (requester_id, recipient_id)
  WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at TIMESTAMPTZ,
  CHECK (user_a <> user_b)
);

CREATE UNIQUE INDEX IF NOT EXISTS matches_unique_pair
  ON matches (LEAST(user_a, user_b), GREATEST(user_a, user_b));

CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID REFERENCES matches(id) ON DELETE CASCADE,
  match_request_id UUID REFERENCES match_requests(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK ((match_id IS NOT NULL) <> (match_request_id IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS messages_match_idx ON messages(match_id, created_at);
CREATE INDEX IF NOT EXISTS messages_match_request_idx ON messages(match_request_id, created_at);

-- Performance indexes from design doc §7
CREATE INDEX IF NOT EXISTS match_requests_recipient_status_idx
    ON match_requests(recipient_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS match_requests_requester_status_idx
    ON match_requests(requester_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS messages_match_request_created_idx
    ON messages(match_request_id, created_at DESC)
    WHERE match_request_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS messages_match_request_created_idx;
DROP INDEX IF EXISTS match_requests_requester_status_idx;
DROP INDEX IF EXISTS match_requests_recipient_status_idx;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS match_requests;
