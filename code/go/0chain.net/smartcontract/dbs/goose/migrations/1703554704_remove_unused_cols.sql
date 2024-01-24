-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS is_available;
ALTER TABLE users DROP COLUMN IF EXISTS change;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS is_available BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS change BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd
