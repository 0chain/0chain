-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS used;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS used bigint default 0;
-- +goose StatementEnd
