-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS name;
ALTER TABLE blobbers DROP COLUMN IF EXISTS website_url;
ALTER TABLE blobbers DROP COLUMN IF EXISTS logo_url;
ALTER TABLE blobbers DROP COLUMN IF EXISTS description;
ALTER TABLE blobbers DROP COLUMN IF EXISTS latitude;
ALTER TABLE blobbers DROP COLUMN IF EXISTS longitude;
ALTER TABLE blobbers DROP COLUMN IF EXISTS used;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS name text default '';
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS website_url text default '';
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS logo_url text default '';
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS description text default '';
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS latitude numeric default 0;
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS longitude numeric default 0;
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS used bigint default 0;
-- +goose StatementEnd