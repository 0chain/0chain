-- +goose Up
-- +goose StatementBegin
ALTER TABLE snapshots ADD COLUMN total_allocations BIGINT NOT NULL DEFAULT 0;

UPDATE snapshots
SET total_allocations = (select count(*) from allocations)
WHERE round = (select max(round) from snapshots);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE snapshots DROP COLUMN total_allocations;
-- +goose StatementEnd
