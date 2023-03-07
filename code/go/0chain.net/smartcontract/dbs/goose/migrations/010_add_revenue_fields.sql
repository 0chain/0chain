-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN total_storage_income bigint NOT NULL DEFAULT 0;
ALTER TABLE blobbers ADD COLUMN total_read_income bigint NOT NULL DEFAULT 0;
ALTER TABLE blobbers ADD COLUMN total_slashed_stake bigint NOT NULL DEFAULT 0;

ALTER TABLE blobber_aggregates ADD COLUMN total_storage_income bigint NOT NULL DEFAULT 0;
ALTER TABLE blobber_aggregates ADD COLUMN total_read_income bigint NOT NULL DEFAULT 0;
ALTER TABLE blobber_aggregates ADD COLUMN total_slashed_stake bigint NOT NULL DEFAULT 0;

ALTER TABLE blobber_snapshots ADD COLUMN total_storage_income bigint NOT NULL DEFAULT 0;
ALTER TABLE blobber_snapshots ADD COLUMN total_read_income bigint NOT NULL DEFAULT 0;
ALTER TABLE blobber_snapshots ADD COLUMN total_slashed_stake bigint NOT NULL DEFAULT 0;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN total_storage_income;
ALTER TABLE blobbers DROP COLUMN total_read_income;
ALTER TABLE blobbers DROP COLUMN total_slashed_stake;

ALTER TABLE blobber_aggregates DROP COLUMN total_storage_income;
ALTER TABLE blobber_aggregates DROP COLUMN total_read_income;
ALTER TABLE blobber_aggregates DROP COLUMN total_slashed_stake;

ALTER TABLE blobber_snapshots DROP COLUMN total_storage_income;
ALTER TABLE blobber_snapshots DROP COLUMN total_read_income;
ALTER TABLE blobber_snapshots DROP COLUMN total_slashed_stake;
-- +goose StatementEnd