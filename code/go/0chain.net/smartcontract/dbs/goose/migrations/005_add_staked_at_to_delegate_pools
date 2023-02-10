-- +goose Up
-- +goose StatementBegin
ALTER TABLE delegate_pools ADD COLUMN IF NOT EXISTS staked_at timestamp with time zone;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE delegate_pools DROP COLUMN IF EXISTS staked_at;
-- +goose StatementEnd