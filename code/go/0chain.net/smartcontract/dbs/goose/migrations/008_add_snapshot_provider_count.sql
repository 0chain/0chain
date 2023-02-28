-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS miner_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS sharder_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS blobber_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS authorizer_count bigint default 0;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS validator_count bigint default 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshots DROP COLUMN IF EXISTS miner_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS sharder_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS blobber_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS authorizer_count;
ALTER TABLE snapshots DROP COLUMN IF EXISTS validator_count;
-- +goose StatementEnd