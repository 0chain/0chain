-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_aggregates ADD COLUMN claimable_reward bigint NOT NULL DEFAULT 0;
ALTER TABLE user_snapshots ADD COLUMN claimable_reward bigint NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_aggregates DROP COLUMN claimable_reward;
ALTER TABLE user_snapshots DROP COLUMN claimable_reward;
-- +goose StatementEnd
