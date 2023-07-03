-- +goose Up
-- +goose StatementBegin
ALTER TABLE challenge_pools DROP COLUMN IF EXISTS allocation_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE challenge_pools CREATE COLUMN IF NOT EXISTS allocation_id text;
-- +goose StatementEnd
