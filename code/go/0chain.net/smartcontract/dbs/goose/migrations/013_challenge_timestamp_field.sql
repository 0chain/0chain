-- +goose Up
-- +goose StatementBegin
ALTER TABLE challenges ADD COLUMN IF NOT EXISTS timestamp bigint;
-- +goose StatementEnd