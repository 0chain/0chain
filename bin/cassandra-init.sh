#!/bin/bash
/0chain/bin/cassandra-wait.sh -t 0 cassandra:9042 -- echo "cassandra started"
cqlsh -f /0chain/sql/zerochain_keyspace.sql cassandra
cqlsh -f /0chain/sql/txn_summary.sql cassandra
echo "cassandra initialized"
