#!/bin/sh
mkdir -p /0chain/miner/data/redis/state
mkdir -p /0chain/miner/data/redis/transactions
mkdir -p /0chain/miner/data/rocksdb
mkdir -p /0chain/miner/log
sudo chown -R root:root /0chain/miner

mkdir -p /0chain/sharder/data/blocks
mkdir -p /0chain/sharder/data/rocksdb
mkdir -p /0chain/sharder/data/cassandra
mkdir -p /0chain/sharder/log
sudo chown -R root:root /0chain/sharder
