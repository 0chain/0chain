-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocations ADD COLUMN is_special_status BOOLEAN DEFAULT FALSE;
ALTER TABLE blobbers ADD COLUMN is_special_status BOOLEAN DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocations DROP COLUMN is_special_status;
ALTER TABLE blobbers DROP COLUMN is_special_status;
-- +goose StatementEnd
