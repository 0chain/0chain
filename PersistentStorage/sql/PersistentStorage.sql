drop database IF EXISTS "0chain";
drop user IF EXISTS miner;

create user miner WITH PASSWORD '0n32b!Ndt43M';
create database "0chain" with owner = miner;

\c 0chain

CREATE TABLE transactions
(
  client_id text NOT NULL,
  transaction_data text NOT NULL,
  createdate timestamp without time zone NOT NULL,
  hash text  PRIMARY KEY,
  signature text NOT NULL
)
WITH ( 
  OIDS=FALSE
);

CREATE TABLE clients
(
  public_key text NOT NULL,
  client_id text PRIMARY KEY
)
WITH (
  OIDS=FALSE
);

ALTER TABLE clients
  OWNER TO miner; 

ALTER TABLE transactions
  OWNER TO miner; 
