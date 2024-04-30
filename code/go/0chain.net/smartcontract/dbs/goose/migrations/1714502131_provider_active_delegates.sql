-- +goose Up
-- +goose StatementBegin
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS active_delegates bigint;
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
ALTER TABLE blobbers ADD COLUMN IF NOT EXISTS num_delegates bigint;


ALTER TABLE validators ADD COLUMN IF NOT EXISTS active_delegates bigint;
UPDATE validators
SET active_delegates = (
    SELECT COUNT(*)
    FROM delegate_pools
    WHERE delegate_pools.provider_id = validators.id
      AND delegate_pools.status = 0
)
WHERE EXISTS (
    SELECT 1
    FROM delegate_pools
    WHERE delegate_pools.provider_id = validators.id
      AND delegate_pools.status = 0
);
ALTER TABLE validators ADD COLUMN IF NOT EXISTS num_delegates bigint;

ALTER TABLE miners ADD COLUMN IF NOT EXISTS active_delegates bigint;
UPDATE miners
SET active_delegates = (
    SELECT COUNT(*)
    FROM delegate_pools
    WHERE delegate_pools.provider_id = miners.id
      AND delegate_pools.status = 0
)
WHERE EXISTS (
    SELECT 1
    FROM delegate_pools
    WHERE delegate_pools.provider_id = miners.id
      AND delegate_pools.status = 0
);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS num_delegates bigint;

ALTER TABLE sharders ADD COLUMN IF NOT EXISTS active_delegates bigint;
UPDATE sharders
SET active_delegates = (
    SELECT COUNT(*)
    FROM delegate_pools
    WHERE delegate_pools.provider_id = sharders.id
      AND delegate_pools.status = 0
)
WHERE EXISTS (
    SELECT 1
    FROM delegate_pools
    WHERE delegate_pools.provider_id = sharders.id
      AND delegate_pools.status = 0
);
ALTER TABLE sharders ADD COLUMN IF NOT EXISTS num_delegates bigint;

ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS active_delegates bigint;
UPDATE authorizers
SET active_delegates = (
    SELECT COUNT(*)
    FROM delegate_pools
    WHERE delegate_pools.provider_id = authorizers.id
      AND delegate_pools.status = 0
)
WHERE EXISTS (
    SELECT 1
    FROM delegate_pools
    WHERE delegate_pools.provider_id = authorizers.id
      AND delegate_pools.status = 0
);
ALTER TABLE authorizers ADD COLUMN IF NOT EXISTS num_delegates bigint;



ALTER TABLE blobber_aggregates ADD COLUMN IF NOT EXISTS active_delegates bigint;
ALTER TABLE validator_aggregates ADD COLUMN IF NOT EXISTS active_delegates bigint;
ALTER TABLE miner_aggregates ADD COLUMN IF NOT EXISTS active_delegates bigint;
ALTER TABLE sharder_aggregates ADD COLUMN IF NOT EXISTS active_delegates bigint;
ALTER TABLE authorizer_aggregates ADD COLUMN IF NOT EXISTS active_delegates bigint;

ALTER TABLE blobber_aggregates ADD COLUMN IF NOT EXISTS num_delegates bigint;
ALTER TABLE validator_aggregates ADD COLUMN IF NOT EXISTS num_delegates bigint;
ALTER TABLE miner_aggregates ADD COLUMN IF NOT EXISTS num_delegates bigint;
ALTER TABLE sharder_aggregates ADD COLUMN IF NOT EXISTS num_delegates bigint;
ALTER TABLE authorizer_aggregates ADD COLUMN IF NOT EXISTS num_delegates bigint;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blobbers DROP COLUMN IF EXISTS active_delegates;
ALTER TABLE validators DROP COLUMN IF EXISTS active_delegates;
ALTER TABLE miners DROP COLUMN IF EXISTS active_delegates;
ALTER TABLE sharders DROP COLUMN IF EXISTS active_delegates;
ALTER TABLE authorizers DROP COLUMN IF EXISTS active_delegates;

ALTER TABLE blobbers DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE validators DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE miners DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE sharders DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE authorizers DROP COLUMN IF EXISTS num_delegates;


ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS active_delegates;
ALTER TABLE validator_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE miner_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE sharder_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE authorizer_aggregates DROP COLUMN IF EXISTS num_delegates;


ALTER TABLE blobber_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE validator_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE miner_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE sharder_aggregates DROP COLUMN IF EXISTS num_delegates;
ALTER TABLE authorizer_aggregates DROP COLUMN IF EXISTS num_delegates;
-- +goose StatementEnd
