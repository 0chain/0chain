-- +goose Up
-- +goose StatementBegin
ALTER SYSTEM SET enable_partitionwise_join = on;
ALTER SYSTEM SET enable_partitionwise_aggregate = on;

CREATE INDEX transactions_client_id_round_hash ON transactions (client_id, round DESC, hash DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER SYSTEM RESET enable_partitionwise_join;
ALTER SYSTEM RESET enable_partitionwise_aggregate;

DROP INDEX transactions_client_id_round_hash;
-- +goose StatementEnd
