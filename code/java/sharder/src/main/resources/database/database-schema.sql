CREATE TABLE public.block
(
 block_hash text NOT NULL,
 prev_block_hash text NOT NULL,
 block_signature text NOT NULL,
 miner_id text NOT NULL,
 “timestamp” timestamp without time zone NOT NULL,
 round integer NOT NULL,
 CONSTRAINT block_pkey PRIMARY KEY (block_hash)
)
WITH (
 OIDS=FALSE
);


CREATE TABLE public.blockTransactions
(
 block_hash text NOT NULL,
 hash_msg text NOT NULL,
 CONSTRAINT blockTransaction_pkey PRIMARY KEY (block_hash, hash_msg)
)
WITH (
 OIDS=FALSE
);

CREATE TABLE public.blockSigners
(
 block_hash text NOT NULL,
 miner_id text NOT NULL,
 CONSTRAINT blocksigners_pkey PRIMARY KEY (block_hash, miner_id)
)
WITH (
 OIDS=FALSE
);