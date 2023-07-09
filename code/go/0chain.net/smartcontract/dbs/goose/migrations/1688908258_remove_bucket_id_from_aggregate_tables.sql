-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobber_aggregates DROP COLUMN bucket_id;
ALTER TABLE miner_aggregates DROP COLUMN bucket_id;
ALTER TABLE sharder_aggregates DROP COLUMN bucket_id;
ALTER TABLE validator_aggregates DROP COLUMN bucket_id;
ALTER TABLE authorizer_aggregates DROP COLUMN bucket_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobber_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE miner_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE sharder_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE validator_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE authorizer_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
-- +goose StatementEnd
