-- +goose Up
-- +goose StatementBegin
UPDATE user_snapshots SET total_reward = original.claimable_rewards+collected_reward FROM
    (SELECT delegate_id, SUM(reward) AS claimable_rewards FROM delegate_pools GROUP BY delegate_id)
        AS original WHERE user_snapshots.user_id = original.delegate_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
