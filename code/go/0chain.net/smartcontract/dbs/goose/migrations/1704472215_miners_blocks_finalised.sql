-- +goose Up
-- +goose StatementBegin
ALTER TABLE miners ADD COLUMN blocks_finalised BIGINT NOT NULL DEFAULT (select count(*) from blocks where miner_id = miners.id);
ALTER TABLE miner_aggregates ADD COLUMN blocks_finalised BIGINT NOT NULL DEFAULT (select count(*) from blocks where miner_id = miner_aggregates.id);
ALTER TABLE miner_snapshots ADD COLUMN blocks_finalised BIGINT NOT NULL DEFAULT (select count(*) from blocks where miner_id = miner_snapshots.id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE miners DROP COLUMN blocks_finalised;
ALTER TABLE miner_aggregates DROP COLUMN blocks_finalised;
ALTER TABLE miner_snapshots DROP COLUMN blocks_finalised;
-- +goose StatementEnd
