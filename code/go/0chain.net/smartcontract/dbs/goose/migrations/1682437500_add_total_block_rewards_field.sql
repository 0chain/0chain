-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN total_block_rewards bigint NOT NULL DEFAULT 0;
ALTER TABLE blobber_aggregates ADD COLUMN total_block_rewards bigint NOT NULL DEFAULT 0;
ALTER TABLE blobber_snapshots ADD COLUMN total_block_rewards bigint NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN total_block_rewards;
ALTER TABLE blobber_aggregates DROP COLUMN total_block_rewards;
ALTER TABLE blobber_snapshots DROP COLUMN total_block_rewards;
-- +goose StatementEnd
