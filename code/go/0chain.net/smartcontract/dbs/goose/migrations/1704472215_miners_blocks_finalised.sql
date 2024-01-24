-- +goose Up
-- +goose StatementBegin

ALTER TABLE miners ADD COLUMN blocks_finalised BIGINT NOT NULL DEFAULT 0;
ALTER TABLE miner_aggregates ADD COLUMN blocks_finalised BIGINT NOT NULL DEFAULT 0;
ALTER TABLE miner_snapshots ADD COLUMN blocks_finalised BIGINT NOT NULL DEFAULT 0;


UPDATE miners
SET blocks_finalised = (SELECT COUNT(*) FROM blocks WHERE miner_id = miners.id);

UPDATE miner_aggregates
SET blocks_finalised = (SELECT COUNT(*) FROM blocks WHERE miner_id = miner_aggregates.miner_id);

UPDATE miner_snapshots
SET blocks_finalised = (SELECT COUNT(*) FROM blocks WHERE miner_id = miner_snapshots.miner_id);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE miners DROP COLUMN blocks_finalised;
ALTER TABLE miner_aggregates DROP COLUMN blocks_finalised;
ALTER TABLE miner_snapshots DROP COLUMN blocks_finalised;
-- +goose StatementEnd
