-- +goose Up
-- +goose StatementBegin

ALTER TABLE blocks ADD COLUMN finality_duration NUMERIC DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE blocks DROP COLUMN finality_duration;

-- +goose StatementEnd
