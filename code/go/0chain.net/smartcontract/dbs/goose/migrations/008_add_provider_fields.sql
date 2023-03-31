-- +goose Up
-- +goose StatementBegin
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS is_killed boolean;
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS is_shutdown boolean;

ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS is_killed boolean;
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS is_shutdown boolean;

ALTER TABLE miners ADD COLUMN IF NOT EXISTS is_killed boolean;
ALTER TABLE miners ADD COLUMN IF NOT EXISTS is_shutdown boolean;

ALTER TABLE sharders ADD COLUMN IF NOT EXISTS is_killed boolean;
ALTER TABLE sharders ADD COLUMN IF NOT EXISTS is_shutdown boolean;

ALTER TABLE validators ADD COLUMN IF NOT EXISTS is_killed boolean;
ALTER TABLE validators ADD COLUMN IF NOT EXISTS is_shutdown boolean;
-- +goose StatementEnd