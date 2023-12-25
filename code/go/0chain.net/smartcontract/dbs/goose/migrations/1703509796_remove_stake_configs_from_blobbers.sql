-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS min_stake;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS max_stake;
-- +goose StatementEnd
