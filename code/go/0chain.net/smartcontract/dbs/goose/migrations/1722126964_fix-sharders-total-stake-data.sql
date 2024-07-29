-- +goose Up
-- +goose StatementBegin
UPDATE sharders
SET total_stake = (
    SELECT SUM(balance)
    FROM delegate_pools
    WHERE status = 0
      AND provider_id = sharders.id
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
