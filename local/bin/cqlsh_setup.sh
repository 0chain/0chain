#!/bin/sh


RepoRoot=$(git rev-parse --show-toplevel)
cqlsh --file $RepoRoot/docker.local/config/cassandra/init.cql

# From bin/cassandra-init.sh
cqlsh --file $RepoRoot/sql/zerochain_keyspace.sql
cqlsh --file $RepoRoot/sql/magic_block_map.sql
# txn_summary is defined in init.cql without a round field so this does nothing
# cqlsh --file $RepoRoot/sql/txn_summary.sql
