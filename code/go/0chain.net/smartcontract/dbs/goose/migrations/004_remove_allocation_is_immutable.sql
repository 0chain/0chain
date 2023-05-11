-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocations DROP COLUMN IF EXISTS is_immutable;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocations CREATE COLUMN IF NOT EXISTS is_immutable boolean default false;
-- +goose StatementEnd