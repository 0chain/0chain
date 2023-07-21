-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE miner_aggregates DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE sharder_aggregates DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE validator_aggregates DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE authorizer_aggregates DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE miner_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE sharder_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE miner_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE validator_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE authorizer_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE miners DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE sharders DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE blobbers DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE validators DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE authorizers DROP COLUMN IF EXISTS bucket_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobber_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE miner_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE sharder_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE validator_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE authorizer_aggregates ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE miner_snapshots ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE sharder_snapshots ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE miner_snapshots ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE validator_snapshots ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE authorizer_snapshots ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE miners ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE sharders ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE blobbers ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE validators ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
ALTER TABLE authorizers ADD COLUMN bucket_id bigint NOT NULL DEFAULT 0;
-- +goose StatementEnd
