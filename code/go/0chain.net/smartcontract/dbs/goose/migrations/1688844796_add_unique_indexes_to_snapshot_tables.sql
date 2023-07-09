-- +goose Up
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_blobber_snapshots_blobber_id;
DROP INDEX IF EXISTS idx_miner_snapshots_miner_id;
DROP INDEX IF EXISTS idx_sharder_snapshots_sharder_id;
DROP INDEX IF EXISTS idx_validator_snapshots_validator_id;
DROP INDEX IF EXISTS idx_authorizer_snapshots_authorizer_id;
CREATE UNIQUE INDEX blobber_snapshots_blobber_id_idx ON blobber_snapshots (blobber_id);
CREATE UNIQUE INDEX miner_snapshots_miner_id_idx ON miner_snapshots (miner_id);
CREATE UNIQUE INDEX sharder_snapshots_sharder_id_idx ON sharder_snapshots (sharder_id);
CREATE UNIQUE INDEX validator_snapshots_validator_id_idx ON validator_snapshots (validator_id);
CREATE UNIQUE INDEX authorizer_snapshots_authorizer_id_idx ON authorizer_snapshots (authorizer_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX blobber_snapshots_blobber_id_idx;
DROP INDEX miner_snapshots_miner_id_idx;
DROP INDEX sharder_snapshots_sharder_id_idx;
DROP INDEX validator_snapshots_validator_id_idx;
DROP INDEX authorizer_snapshots_authorizer_id_idx;
CREATE INDEX idx_blobber_snapshots_blobber_id ON blobber_snapshots (blobber_id);
CREATE INDEX idx_miner_snapshots_miner_id ON miner_snapshots (miner_id);
CREATE INDEX idx_sharder_snapshots_sharder_id ON sharder_snapshots (sharder_id);
CREATE INDEX idx_validator_snapshots_validator_id ON validator_snapshots (validator_id);
CREATE INDEX idx_authorizer_snapshots_authorizer_id ON authorizer_snapshots (authorizer_id);
-- +goose StatementEnd
