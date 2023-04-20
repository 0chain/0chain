-- +goose Up
-- +goose StatementBegin
ALTER TABLE authorizers DROP COLUMN IF EXISTS third_party_extendable;
ALTER TABLE blobbers DROP COLUMN IF EXISTS third_party_extendable;
ALTER TABLE miners DROP COLUMN IF EXISTS third_party_extendable;
ALTER TABLE sharders DROP COLUMN IF EXISTS third_party_extendable;
ALTER TABLE validators DROP COLUMN IF EXISTS third_party_extendable;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
