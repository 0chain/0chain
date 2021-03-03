#!/bin/sh

RepoRoot=$(git rev-parse --show-toplevel)
cqlsh> --file $RepoRoot/local/config/cassandra/init.cql
cqlsh> --file $RepoRoot/sql/zerochain_keyspace.sql
cqlsh> --file $RepoRoot/sql/magic_block_map.sql
cqlsh> --file $RepoRoot/sql/txn_summary.sql