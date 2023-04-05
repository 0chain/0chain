-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS is_available boolean;
-- +goose StatementEnd