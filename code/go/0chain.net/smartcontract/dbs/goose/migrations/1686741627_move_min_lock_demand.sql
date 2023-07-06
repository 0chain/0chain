-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocations ADD COLUMN min_lock_demand numeric;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocation_blobber_terms DROP COLUMN min_lock_demand;
ALTER TABLE blobbers DROP COLUMN min_lock_demand;
-- +goose StatementEnd