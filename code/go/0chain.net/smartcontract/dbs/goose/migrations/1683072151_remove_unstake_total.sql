-- +goose Up
-- +goose StatementBegin
ALTER TABLE authorizer_aggregates DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE authorizer_snapshots DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE authorizers DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE blobber_snapshots DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE blobbers DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE miner_aggregates DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE miner_snapshots DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE miners DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE sharder_aggregates DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE sharder_snapshots DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE sharders DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE validator_aggregates DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE validator_snapshots DROP COLUMN IF EXISTS unstake_total;
ALTER TABLE validators DROP COLUMN IF EXISTS unstake_total;
-- +goose StatementEnd