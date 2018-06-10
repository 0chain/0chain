CREATE TABLE IF NOT EXISTS zerochain.block_summary (
hash text PRIMARY KEY,
version text,
round bigint,
merkle_root text,
miner_id text,
creation_date bigint
);

CREATE INDEX IF NOT EXISTS block_summary_nu1_creation_date ON zerochain.block_summary (creation_date);
CREATE INDEX IF NOT EXISTS block_summary_u1_round ON zerochain.block_summary (round);
