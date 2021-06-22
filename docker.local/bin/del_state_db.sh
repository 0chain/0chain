#!/bin/sh

for i in $(seq 1 3)
do
  rm -rf docker.local/miner$i/data/rocksdb/state
done

for i in $(seq 1 3)
do
  rm -rf docker.local/sharder$i/data/rocksdb/state
done
