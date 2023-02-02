-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS third_party_extendable boolean default false;
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS file_options smallint default 63;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocations DROP COLUMN IF EXISTS third_party_extendable;
ALTER TABLE allocations DROP COLUMN IF EXISTS file_options;
-- +goose StatementEnd