-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobber_aggregates ADD COLUMN IF NOT EXISTS is_restricted boolean;
ALTER TABLE blobber_aggregates ADD COLUMN IF NOT EXISTS not_available boolean;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS is_restricted;
ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS not_available;
-- +goose StatementEnd
