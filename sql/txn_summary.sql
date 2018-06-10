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
