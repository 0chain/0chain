#!/bin/bash
/0chain/bin/wait-for-service.sh -t 0 cassandra:9042 -- echo "cassandra started"
cqlsh -f /0chain/sql/zerochain_keyspace.sql cassandra
cqlsh -f /0chain/sql/magic_block_map.sql cassandra
cqlsh -f /0chain/sql/txn_summary.sql cassandra
echo "cassandra initialized"
