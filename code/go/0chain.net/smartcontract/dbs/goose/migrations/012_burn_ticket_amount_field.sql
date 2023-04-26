-- +goose Up
-- +goose StatementBegin
ALTER TABLE burn_tickets ADD COLUMN IF NOT EXISTS amount bigint;
-- +goose StatementEnd