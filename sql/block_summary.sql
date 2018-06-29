CREATE TABLE IF NOT EXISTS zerochain.block_summary (
hash text,
round bigint,
creation_date bigint,
version text,
round_random_seed bigint,
merkle_tree_root text,
PRIMARY KEY(hash)
);
