-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_miner_creation_round ON miners (creation_round);
CREATE INDEX idx_sharder_creation_round ON sharders (creation_round);
CREATE INDEX idx_wblocknum ON write_markers (block_number);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
