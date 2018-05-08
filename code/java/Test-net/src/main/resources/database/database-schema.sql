CREATE TABLE public.clients
(
  public_key text NOT NULL,
  hash_key text NOT NULL,
  CONSTRAINT clients_pkey PRIMARY KEY (hash_key)
)
  WITH (
  OIDS=FALSE
);

CREATE TABLE public.transaction
(
  client_id text NOT NULL,
  data text NOT NULL,
  "timestamp" timestamp without time zone NOT NULL,
  hash_msg text NOT NULL,
  sign text NOT NULL,
  CONSTRAINT transaction_pkey PRIMARY KEY (hash_msg)
)
WITH (
  OIDS=FALSE
);

CREATE INDEX transaction_cliend_id_idx
  ON public.transaction
  USING btree
  (client_id COLLATE pg_catalog."default");