-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_rew_block_prov;
DROP INDEX IF EXISTS idx_rew_del_prov;
DROP INDEX IF EXISTS idx_event;
DROP INDEX IF EXISTS idx_tcreation_date;
DROP INDEX IF EXISTS idx_bcreation_date;
DROP INDEX IF EXISTS idx_challenges_round_responded;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
