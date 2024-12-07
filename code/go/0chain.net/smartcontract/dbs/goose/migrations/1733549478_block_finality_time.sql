-- +goose Up
-- +goose StatementBegin

ALTER TABLE blocks ADD COLUMN finality_time NUMERIC DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE blocks DROP COLUMN finality_time;

-- +goose StatementEnd
