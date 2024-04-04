-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_aggregates RENAME COLUMN claimable_reward TO total_reward;
ALTER TABLE user_snapshots RENAME COLUMN claimable_reward TO total_reward;

UPDATE user_snapshots SET total_reward = original.total_reward FROM 
    (SELECT delegate_id, SUM(total_reward) AS total_reward FROM delegate_pools GROUP BY delegate_id)
AS original WHERE user_snapshots.user_id = original.delegate_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_aggregates RENAME COLUMN total_reward TO claimable_reward;
ALTER TABLE user_snapshots RENAME COLUMN total_reward TO claimable_reward;
-- +goose StatementEnd
