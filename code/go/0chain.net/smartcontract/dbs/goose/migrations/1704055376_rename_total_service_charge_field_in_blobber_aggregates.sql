-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobber_aggregates DROP COLUMN total_service_charge;
ALTER TABLE blobber_aggregates ADD COLUMN service_charge real NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobber_aggregates ADD COLUMN total_service_charge BIGINT NOT NULL DEFAULT 0;
ALTER TABLE blobber_aggregates DROP COLUMN service_charge;
-- +goose StatementEnd
