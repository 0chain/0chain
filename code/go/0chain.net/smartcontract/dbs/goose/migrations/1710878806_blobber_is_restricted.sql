-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS is_restricted bool DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS is_restricted;
-- +goose StatementEnd
