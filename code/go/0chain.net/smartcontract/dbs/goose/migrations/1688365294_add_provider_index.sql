-- +goose Up
-- +goose StatementBegin
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS idx int;
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS idx int;

ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS idx int;
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS idx int;

ALTER TABLE miners ADD COLUMN IF NOT EXISTS idx int;
ALTER TABLE miners ADD COLUMN IF NOT EXISTS idx int;

ALTER TABLE sharders ADD COLUMN IF NOT EXISTS idx int;
ALTER TABLE sharders ADD COLUMN IF NOT EXISTS idx int;

ALTER TABLE validators ADD COLUMN IF NOT EXISTS idx int;
ALTER TABLE validators ADD COLUMN IF NOT EXISTS idx int;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
