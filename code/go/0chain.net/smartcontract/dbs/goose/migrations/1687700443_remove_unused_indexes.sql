-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_rew_block_prov;
DROP INDEX IF EXISTS idx_rew_del_prov;
DROP INDEX IF EXISTS idx_event;
DROP INDEX IF EXISTS idx_tcreation_date;
DROP INDEX IF EXISTS idx_bcreation_date;
DROP INDEX IF EXISTS idx_challenges_round_responded;
DROP INDEX IF EXISTS idx_challenges_deleted_at;
ALTER TABLE challenges DROP COLUMN deleted_at;
DROP INDEX IF EXISTS idx_ba_rankmetric;
DROP INDEX IF EXISTS idx_walloc_file;
DROP INDEX IF EXISTS idx_wblocknum;
DROP INDEX IF EXISTS idx_astart_time;
DROP INDEX IF EXISTS idx_authorizer_creation_round;
DROP INDEX IF EXISTS idx_authorizer_snapshots_creation_round;
DROP INDEX IF EXISTS idx_validator_creation_round;
DROP INDEX IF EXISTS idx_validator_snapshots_creation_round;
DROP INDEX IF EXISTS idx_miner_creation_round;
DROP INDEX IF EXISTS idx_miner_snapshots_creation_round;
DROP INDEX IF EXISTS idx_sharder_creation_round;
DROP INDEX IF EXISTS idx_sharder_snapshots_creation_round;
DROP INDEX IF EXISTS idx_blobber_creation_round;
DROP INDEX IF EXISTS idx_blobber_snapshots_creation_round;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
