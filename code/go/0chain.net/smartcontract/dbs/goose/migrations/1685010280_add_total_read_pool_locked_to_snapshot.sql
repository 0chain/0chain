-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN total_read_pool_locked BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshot DROP COLUMN total_read_pool_locked;
-- +goose StatementEnd
