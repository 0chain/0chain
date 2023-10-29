-- +goose Up
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN is_published BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE events ADD COLUMN sequence_number BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE events DROP COLUMN is_published;
ALTER TABLE events DROP COLUMN sequence_number;
-- +goose StatementEnd
