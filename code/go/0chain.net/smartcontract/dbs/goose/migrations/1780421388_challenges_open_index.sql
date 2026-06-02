-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_copen_challenge_round ON challenges (blobber_id, responded, round_created_at);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_copen_challenge_round;
