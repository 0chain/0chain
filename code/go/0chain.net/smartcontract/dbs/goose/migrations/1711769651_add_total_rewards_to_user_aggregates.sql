-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_aggregates ADD COLUMN total_reward BIGINT DEFAULT 0;
ALTER TABLE user_snapshots ADD COLUMN total_reward BIGINT DEFAULT 0;
ALTER TABLE user_aggregates DROP COLUMN claimable_reward;
ALTER TABLE user_snapshots DROP COLUMN claimable_reward;

UPDATE user_snapshots SET total_reward = original.total_reward FROM 
    (SELECT delegate_id, SUM(total_reward) AS total_reward FROM delegate_pools GROUP BY delegate_id)
AS original WHERE user_snapshots.user_id = original.delegate_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_aggregates DROP COLUMN total_reward;
ALTER TABLE user_snapshots DROP COLUMN total_reward;
ALTER TABLE user_aggregates ADD COLUMN claimable_reward BIGINT DEFAULT 0;
ALTER TABLE user_snapshots ADD COLUMN claimable_reward BIGINT DEFAULT 0;
-- +goose StatementEnd
