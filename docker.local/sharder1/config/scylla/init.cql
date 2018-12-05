CREATE  KEYSPACE IF NOT EXISTS zerochain
WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }
AND DURABLE_WRITES = true;

CREATE TABLE IF NOT EXISTS zerochain.txn_summary (
hash text PRIMARY KEY,
version text,
block_hash text,
creation_date bigint,
client_id text,
to_client_id text
);

CREATE INDEX IF NOT EXISTS txn_summary_nu1_creation_date ON zerochain.txn_summary (creation_date);
CREATE INDEX IF NOT EXISTS txn_summary_nu2_client_id ON zerochain.txn_summary (client_id);
CREATE INDEX IF NOT EXISTS txn_summary_nu3_to_client_id ON zerochain.txn_summary (to_client_id);


CREATE TABLE IF NOT EXISTS zerochain.block_summary (
hash text,
prev_hash text,
round bigint,
creation_date bigint,
version text,
round_random_seed bigint,
PRIMARY KEY(hash, round)
);

CREATE INDEX IF NOT EXISTS block_summary_nu1_creation_date ON zerochain.block_summary (creation_date);
CREATE INDEX IF NOT EXISTS block_summary_u1_round ON zerochain.block_summary (round);
