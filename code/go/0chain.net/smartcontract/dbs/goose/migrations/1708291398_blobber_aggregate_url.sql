-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobber_aggregates ADD COLUMN url text;
ALTER TABLE blobber_aggregates ADD COLUMN is_killed boolean;
ALTER TABLE blobber_aggregates ADD COLUMN is_shutdown boolean;
ALTER TABLE validator_aggregates ADD COLUMN url text;
ALTER TABLE validator_aggregates ADD COLUMN is_killed boolean;
ALTER TABLE validator_aggregates ADD COLUMN is_shutdown boolean;
ALTER TABLE miner_aggregates ADD COLUMN url text;
ALTER TABLE miner_aggregates ADD COLUMN is_killed boolean;
ALTER TABLE miner_aggregates ADD COLUMN is_shutdown boolean;
ALTER TABLE sharder_aggregates ADD COLUMN url text;
ALTER TABLE sharder_aggregates ADD COLUMN is_killed boolean;
ALTER TABLE sharder_aggregates ADD COLUMN is_shutdown boolean;
ALTER TABLE authorizer_aggregates ADD COLUMN url text;
ALTER TABLE authorizer_aggregates ADD COLUMN is_killed boolean;
ALTER TABLE authorizer_aggregates ADD COLUMN is_shutdown boolean;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobber_aggregates DROP COLUMN url;
ALTER TABLE blobber_aggregates DROP COLUMN is_killed;
ALTER TABLE blobber_aggregates DROP COLUMN is_shutdown;
ALTER TABLE validator_aggregates DROP COLUMN url;
ALTER TABLE validator_aggregates DROP COLUMN is_killed;
ALTER TABLE validator_aggregates DROP COLUMN is_shutdown;
ALTER TABLE miner_aggregates DROP COLUMN url;
ALTER TABLE miner_aggregates DROP COLUMN is_killed;
ALTER TABLE miner_aggregates DROP COLUMN is_shutdown;
ALTER TABLE sharder_aggregates DROP COLUMN url;
ALTER TABLE sharder_aggregates DROP COLUMN is_killed;
ALTER TABLE sharder_aggregates DROP COLUMN is_shutdown;
ALTER TABLE authorizer_aggregates DROP COLUMN url;
ALTER TABLE authorizer_aggregates DROP COLUMN is_killed;
ALTER TABLE authorizer_aggregates DROP COLUMN is_shutdown;

-- +goose StatementEnd
