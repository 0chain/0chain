CREATE TABLE IF NOT EXISTS zerochain.txn_summary (
hash text PRIMARY KEY,
version text,
block_hash text,
creation_date bigint,
);

