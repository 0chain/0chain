#!/bin/sh

service cassandra stop
rm -rf /var/lib/cassandra/*
service cassandra start
"$(dirname "$0")"/../../bin/wait-for-service.sh -t 0 "Test Cluster":9042 -- echo "cassandra started"
#cqlsh -f "$(dirname "$0")"/../../sql/zerochain_keyspace.sql cassandra
#cqlsh -f "$(dirname "$0")"/../../sql/magic_block_map.sql cassandra
#cqlsh -f "$(dirname "$0")"/../../sql/txn_summary.sql cassandra
#echo "cassandra initialized"
#cqlsh -f "$(dirname "$0")"/../../docker.local/config/cassandra/init.cql

# cqlsh --file sql/zerochain_keyspace.sql cassandra
# cqlsh --file sql/magic_block_map.sql cassandra
# cqlsh --file sql/txn_summary.sql cassandra