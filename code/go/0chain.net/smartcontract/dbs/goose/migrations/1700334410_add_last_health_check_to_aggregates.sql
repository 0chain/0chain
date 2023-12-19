-- +goose Up
-- +goose StatementBegin
ALTER TABLE miner_aggregates ADD COLUMN last_health_check BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sharder_aggregates ADD COLUMN last_health_check BIGINT NOT NULL DEFAULT 0;
ALTER TABLE blobber_aggregates ADD COLUMN last_health_check BIGINT NOT NULL DEFAULT 0;
ALTER TABLE validator_aggregates ADD COLUMN last_health_check BIGINT NOT NULL DEFAULT 0;
ALTER TABLE authorizer_aggregates ADD COLUMN last_health_check BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE miner_aggregates DROP COLUMN last_health_check;
ALTER TABLE sharder_aggregates DROP COLUMN last_health_check;
ALTER TABLE blobber_aggregates DROP COLUMN last_health_check;
ALTER TABLE validator_aggregates DROP COLUMN last_health_check;
ALTER TABLE authorizer_aggregates DROP COLUMN last_health_check;
-- +goose StatementEnd
