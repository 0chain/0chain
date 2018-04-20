# Java Test-net

## Requirements

Building the API miner war requires [Maven](https://maven.apache.org/), [PostgreSQL](https://www.postgresql.org), and [Tomcat](https://tomcat.apache.org) to be installed. 

***Please add swagger-codegen-cli.jar to this code/java/bin directory AND apache-maven-3.5.3 to /opt directory***
The build script assumes that apache-maven-3.5.3 is installed in the opt directory, and that swagger-codegen-cli.jar is in the bin directory (same one as the script).

### Assumptions

PostgresSQL is running locally on port 5432 and Tomcat is running locally on port 8080.

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
  sign text NOT NULL,
  CONSTRAINT clients_pkey PRIMARY KEY (hash_key)
)
WITH (
  OIDS=FALSE
);
ALTER TABLE public.clients
  OWNER TO miner;
```

## Installation
For the first time building the modules the following command should be used
```
./build.sh build all
```

If the swagger YAML files for the modules change you can rebuild them with the swagger-codegen by typing one of the following...
```
./build.sh build registration

./build.sh build transaction
```

To update the utils used by the registration and transaction server type
```
./build.sh build utils
```

To update integration tests type
```
./build.sh build integrationTest
```

If only the implementations of the business logic has changed type one of the three commands for the appropriate files that have been changed
```
./build.sh update all

./build.sh update registration

./build.sh update transaction
```

## Deployment

The build script creates a new build directory with sub directory for each module.
The war files for the servers are located in the build directory...
	/transactionServer/target/transaction.war
	/registrationServer/target/registration.war
and can be deployed in a Tomcat server.

## Testing
Test Regsitration Server
```
./build.sh test registration
```

Test Transaction Server
```
./build.sh test transacton
```

Test Both
```
./build.sh test all
```
