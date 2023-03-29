-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN storage_token_stake BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshots DROP COLUMN storage_token_stake;
-- +goose StatementEnd
