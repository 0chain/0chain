-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots DROP COLUMN mined_total;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN mined_total bigint NOT NULL DEFAULT 0;
-- +goose StatementEnd
