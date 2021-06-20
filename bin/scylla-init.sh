#!/bin/bash
/0chain/bin/wait-for-service.sh -t 0 scylla:9042 -- echo "scylla started"
cqlsh -f /0chain/sql/zerochain_keyspace.sql scylla
cqlsh -f /0chain/sql/txn_summary.sql scylla
echo "scylla initialized"
