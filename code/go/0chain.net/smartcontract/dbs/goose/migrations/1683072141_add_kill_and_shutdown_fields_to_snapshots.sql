-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobber_snapshots ADD COLUMN is_killed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE blobber_snapshots ADD COLUMN is_shutdown BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE miner_snapshots ADD COLUMN is_killed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE miner_snapshots ADD COLUMN is_shutdown BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE sharder_snapshots ADD COLUMN is_killed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE sharder_snapshots ADD COLUMN is_shutdown BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE validator_snapshots ADD COLUMN is_killed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE validator_snapshots ADD COLUMN is_shutdown BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE authorizer_snapshots ADD COLUMN is_killed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE authorizer_snapshots ADD COLUMN is_shutdown BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE blobber_snapshots DROP COLUMN is_killed;
ALTER TABLE blobber_snapshots DROP COLUMN is_shutdown;
ALTER TABLE miner_snapshots DROP COLUMN is_killed;
ALTER TABLE miner_snapshots DROP COLUMN is_shutdown;
ALTER TABLE sharder_snapshots DROP COLUMN is_killed;
ALTER TABLE sharder_snapshots DROP COLUMN is_shutdown;
ALTER TABLE validator_snapshots DROP COLUMN is_killed;
ALTER TABLE validator_snapshots DROP COLUMN is_shutdown;
ALTER TABLE authorizer_snapshots DROP COLUMN is_killed;
ALTER TABLE authorizer_snapshots DROP COLUMN is_shutdown;
-- +goose StatementBegin

-- +goose StatementEnd
