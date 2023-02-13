-- +goose Up
-- +goose StatementBegin
ALTER TABLE reward_delegates ADD COLUMN IF NOT EXISTS provider_id text;
-- +goose StatementEnd