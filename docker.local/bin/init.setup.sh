#!/bin/sh

for i in $(seq 1 3)
do
  mkdir -p docker.local/miner$i/data/redis/state
  mkdir -p docker.local/miner$i/data/redis/transactions
  mkdir -p docker.local/miner$i/data/rocksdb
  mkdir -p docker.local/miner$i/log
done

for i in $(seq 1 3)
do
  mkdir -p docker.local/sharder$i/data/blocks
  mkdir -p docker.local/sharder$i/data/rocksdb
  mkdir -p docker.local/sharder$i/data/cassandra
  mkdir -p docker.local/sharder$i/log
done

for i in $(seq 1 1)
do
  mkdir -p docker.local/wallet$i/data/keys
  mkdir -p docker.local/wallet$i/log
done