# Java Test-net

## Requirements

Building the API miner war requires [Maven](https://maven.apache.org/), [PostgreSQL](https://www.postgresql.org), and [Tomcat](https://tomcat.apache.org) to be installed. 

***Please add apache-maven-3.5.3 to /opt directory.***
The build script assumes that apache-maven-3.5.3 is installed in the opt directory.

### Assumptions

PostgresSQL is running locally on port 5432.

### Postgresql 

The database and tables can be created by coping and pasting this into a postgres terminal with a user with privilege
```
create user miner WITH PASSWORD '0n32b!Ndt43M';

create database "0chain" with owner = miner;

\connect 0chain;

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
ALTER TABLE public.transaction
  OWNER TO miner;

CREATE INDEX transaction_cliend_id_idx
  ON public.transaction
  USING btree
  (client_id COLLATE pg_catalog."default");

CREATE TABLE public.clients
(
  public_key text NOT NULL,
  hash_key text NOT NULL,
  CONSTRAINT clients_pkey PRIMARY KEY (hash_key)
)
WITH (
  OIDS=FALSE
);
ALTER TABLE public.clients
  OWNER TO miner;
```

## Installation
First install the Utils module. CD into the Utils directory and use maven clean install
```
mvn clean install
```
Next, cd into the Test-net directory and type the following to run the modules.
```
mvn spring-boot:run
```

## Testing

To run the integration tests type:
```
mvn clean test
```
