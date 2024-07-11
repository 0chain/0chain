-- +goose Up
-- +goose StatementBegin
DROP TABLE miner_aggregates;
DROP TABLE sharder_aggregates;
DROP TABLE blobber_aggregates;
DROP TABLE validator_aggregates;
DROP TABLE authorizer_aggregates;
DROP TABLE user_aggregates;

DROP TABLE miner_snapshots;
DROP TABLE sharder_snapshots;
DROP TABLE blobber_snapshots;
DROP TABLE validator_snapshots;
DROP TABLE authorizer_snapshots;
DROP TABLE user_snapshots;

DROP TABLE snapshots;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
