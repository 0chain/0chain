#!/bin/bash
chmod +x /0chain/cassandra/wait-for-it.sh

/0chain/cassandra/wait-for-it.sh -t 0 cassandra:9042 -- echo "CASSANDRA Node started"

cd ../0chain/sql

cqlsh -f zerochain_keyspace.sql cassandra
cqlsh -f block_summary.sql cassandra
cqlsh -f txn_summary.sql cassandra

echo "### CASSANDRA INITIALISED! ###"
