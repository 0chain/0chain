-- +goose Up
-- +goose StatementBegin
ALTER TABLE blocks DROP COLUMN is_finalised
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blocks ADD COLUMN is_finalised boolean NOT NULL DEFAULT false;
-- +goose StatementEnd
