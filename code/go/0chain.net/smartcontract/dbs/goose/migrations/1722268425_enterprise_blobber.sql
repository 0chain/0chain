-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocations ADD COLUMN is_enterprise BOOLEAN DEFAULT FALSE;
ALTER TABLE blobbers ADD COLUMN is_enterprise BOOLEAN DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocations DROP COLUMN is_enterprise;
ALTER TABLE blobbers DROP COLUMN is_enterprise;
-- +goose StatementEnd
