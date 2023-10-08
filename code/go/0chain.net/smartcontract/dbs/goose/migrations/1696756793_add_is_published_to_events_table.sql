-- +goose Up
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN is_published BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE events DROP COLUMN is_published;
-- +goose StatementEnd
