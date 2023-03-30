-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN miner_total_rewards BIGINT NOT NULL DEFAULT 0;
ALTER TABLE snapshots ADD COLUMN sharder_total_rewards BIGINT NOT NULL DEFAULT 0;
ALTER TABLE snapshots ADD COLUMN blobber_total_rewards BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshots DROP COLUMN miner_total_rewards;
ALTER TABLE snapshots DROP COLUMN sharder_total_rewards;
ALTER TABLE snapshots DROP COLUMN blobber_total_rewards;
-- +goose StatementEnd
