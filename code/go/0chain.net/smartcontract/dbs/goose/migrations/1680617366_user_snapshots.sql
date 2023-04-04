-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_snapshots (
    user_id text,
    round bigint,
    collected_reward bigint,
    total_stake bigint,
    read_pool_total bigint,
    write_pool_total bigint,
    payed_fees bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
)

CREATE UNIQUE INDEX idx_user_snapshots ON user_snapshots USING btree (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_snapshots;
DROP INDEX idx_user_snapshots;
-- +goose StatementEnd
