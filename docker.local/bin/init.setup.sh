#!/bin/sh

mkdir -p docker.local/miner/data/redis/state
mkdir -p docker.local/miner/data/redis/transactions
mkdir -p docker.local/miner/data/rocksdb
mkdir -p docker.local/miner/log

mkdir -p docker.local/benchmarks
for i in $(seq 1 8)
do
  mkdir -p docker.local/miner"$i"/data/redis/state
  mkdir -p docker.local/miner"$i"/data/redis/transactions
  mkdir -p docker.local/miner"$i"/data/rocksdb
  mkdir -p docker.local/miner"$i"/log
done

for i in $(seq 1 4)
do
  mkdir -p docker.local/sharder"$i"/data/blocks
  mkdir -p docker.local/sharder"$i"/data/rocksdb
  mkdir -p docker.local/sharder"$i"/data/cassandra
  mkdir -p docker.local/sharder"$i"/config/cassandra
  cp config/cassandra/* docker.local/sharder"$i"/config/cassandra/.
  mkdir -p docker.local/sharder"$i"/log
  mkdir -p docker.local/sharder"$i"/data/postgresql
done
