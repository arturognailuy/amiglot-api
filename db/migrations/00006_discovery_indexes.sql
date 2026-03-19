-- +goose Up
CREATE TABLE IF NOT EXISTS user_blocks (
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

-- Speed up the supply/demand/bridge subqueries
CREATE INDEX IF NOT EXISTS user_languages_target_idx
    ON user_languages(user_id, language_code, level) WHERE is_target = true;

-- Speed up block lookups
CREATE INDEX IF NOT EXISTS user_blocks_pair_idx
    ON user_blocks(blocker_id, blocked_id);
CREATE INDEX IF NOT EXISTS user_blocks_reverse_idx
    ON user_blocks(blocked_id, blocker_id);

-- +goose Down
DROP INDEX IF EXISTS user_blocks_reverse_idx;
DROP INDEX IF EXISTS user_blocks_pair_idx;
DROP INDEX IF EXISTS user_languages_target_idx;
DROP TABLE IF EXISTS user_blocks;
