-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_read_markers_transaction_id;
DROP INDEX IF EXISTS idx_write_markers_transaction_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
