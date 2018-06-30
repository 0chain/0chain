#!/bin/bash
chmod +x /0chain/cassandra/wait-for-it.sh

/0chain/cassandra/wait-for-it.sh -t 0 cassandra:9042 -- echo "CASSANDRA Node started"

cd ../0chain/sql

cqlsh -f zerochain_keyspace.cql cassandra
cqlsh -f block_summary.cql cassandra
cqlsh -f txn_summary.cql cassandra

echo "### CASSANDRA INITIALISED! ###"
