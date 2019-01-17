CREATE TABLE IF NOT EXISTS zerochain.txn_summary (
hash text PRIMARY KEY,
block_hash text,
round bigint
);
