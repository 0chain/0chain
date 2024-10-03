-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocations ADD COLUMN storage_version text default '';
ALTER TABLE blobbers ADD COLUMN storage_version text default '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocations DROP COLUMN storage_version;
ALTER TABLE blobbers DROP COLUMN storage_version;
-- +goose StatementEnd
