CREATE TABLE public.clients
(
  public_key text NOT NULL,
  clientId text PRIMARY KEY
)
  WITH (
  OIDS=FALSE
);

CREATE TABLE public.transactions
(
  clientId text NOT NULL,
  data text NOT NULL,
  "timestamp" timestamp without time zone NOT NULL,
  hash_msg text PRIMARY KEY,
  sign text NOT NULL,
  status text NOT NULL DEFAULT 'free'::text
)
WITH (
  OIDS=FALSE
);

CREATE INDEX transaction_cliend_id_idx
  ON public.transactions
  USING btree
  (clientId COLLATE pg_catalog."default");

CREATE TABLE public.block
(
  block_hash text PRIMARY KEY,
  prev_block_hash text NOT NULL,
  block_signature text NOT NULL,
  miner_id text NOT NULL,
  round integer NOT NULL
)
WITH (
  OIDS=FALSE
);

CREATE TABLE blocktransactions
(
  hash_msg text NOT NULL,
  block_hash text NOT NULL,
  CONSTRAINT blocktransactions_pkey PRIMARY KEY (hash_msg, block_hash),
  CONSTRAINT block_hash FOREIGN KEY (block_hash)
      REFERENCES block (block_hash) MATCH SIMPLE
      ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT transaction_hash FOREIGN KEY (hash_msg)
      REFERENCES transactions (hash_msg) MATCH SIMPLE
      ON UPDATE NO ACTION ON DELETE NO ACTION
)
WITH (
  OIDS=FALSE
);