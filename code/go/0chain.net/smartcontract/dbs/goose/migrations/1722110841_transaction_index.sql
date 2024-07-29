-- +goose Up
-- +goose StatementBegin

create index transactions_client_id_round_hash on transactions (client_id,round desc, hash desc );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop index transactions_client_id_round_hash;

-- +goose StatementEnd
