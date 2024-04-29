-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS active_delegates boolean;
UPDATE blobbers
SET active_delegates = (
    SELECT COUNT(*)
    FROM delegate_pools
    WHERE delegate_pools.provider_id = blobbers.id
      AND delegate_pools.status = 0
)
WHERE EXISTS (
    SELECT 1
    FROM delegate_pools
    WHERE delegate_pools.provider_id = blobbers.id
      AND delegate_pools.status = 0
);

ALTER TABLE blobber_aggregates ADD COLUMN IF NOT EXISTS active_delegates boolean;
UPDATE blobber_aggregates
SET active_delegates = (
    SELECT COUNT(*)
    FROM delegate_pools
    WHERE delegate_pools.provider_id = blobber_aggregates.id
      AND delegate_pools.status = 0
)
WHERE EXISTS (
    SELECT 1
    FROM delegate_pools
    WHERE delegate_pools.provider_id = blobber_aggregates.id
      AND delegate_pools.status = 0
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS active_delegates;
ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS active_delegates;
-- +goose StatementEnd
