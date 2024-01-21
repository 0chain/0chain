-- +goose Up
-- +goose StatementBegin
ALTER TABLE miners DROP COLUMN IF EXISTS min_stake;
ALTER TABLE miners DROP COLUMN IF EXISTS max_stake;
ALTER TABLE sharders DROP COLUMN IF EXISTS min_stake;
ALTER TABLE sharders DROP COLUMN IF EXISTS max_stake;
ALTER TABLE blobbers DROP COLUMN IF EXISTS min_stake;
ALTER TABLE blobbers DROP COLUMN IF EXISTS max_stake;
ALTER TABLE validators DROP COLUMN IF EXISTS min_stake;
ALTER TABLE validators DROP COLUMN IF EXISTS max_stake;
ALTER TABLE authorizers DROP COLUMN IF EXISTS min_stake;
ALTER TABLE authorizers DROP COLUMN IF EXISTS max_stake;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE miners ADD COLUMN IF NOT EXISTS min_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE miners ADD COLUMN IF NOT EXISTS max_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE sharders ADD COLUMN IF NOT EXISTS min_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE sharders ADD COLUMN IF NOT EXISTS max_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS min_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS max_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE validators ADD COLUMN IF NOT EXISTS min_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE validators ADD COLUMN IF NOT EXISTS max_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS min_stake bigint NOT NULL DEFAULT 0;
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS max_stake bigint NOT NULL DEFAULT 0;
-- +goose StatementEnd
