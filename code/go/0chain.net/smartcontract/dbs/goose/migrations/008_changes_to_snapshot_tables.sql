-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS miner_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS sharder_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS blobber_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS authorizer_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS validator_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS total_txn_fee bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS blobbers_stake bigint default 0;
ALTER TABLE snapshots DROP COLUMN IF EXISTS average_txn_fee;
ALTER TABLE snapshots DROP COLUMN IF EXISTS average_write_price;

ALTER TABLE blobber_snapshots ADD COLUMN IF NOT EXISTS bucket_id bigint default 0;
ALTER TABLE authorizer_snapshots ADD COLUMN IF NOT EXISTS bucket_id bigint default 0;
ALTER TABLE miner_snapshots ADD COLUMN IF NOT EXISTS bucket_id bigint default 0;
ALTER TABLE sharder_snapshots ADD COLUMN IF NOT EXISTS bucket_id bigint default 0;
ALTER TABLE validator_snapshots ADD COLUMN IF NOT EXISTS bucket_id bigint default 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshots DROP COLUMN IF EXISTS miner_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS sharder_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS blobber_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS authorizer_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS validator_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS blobbers_stake;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS average_txn_fee bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS average_write_price bigint default 0;

ALTER TABLE blobber_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE authorizer_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE miner_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE sharder_snapshots DROP COLUMN IF EXISTS bucket_id;
ALTER TABLE validator_snapshots DROP COLUMN IF EXISTS bucket_id;
-- +goose StatementEnd