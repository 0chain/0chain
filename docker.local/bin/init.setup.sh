#!/bin/sh

for i in $(seq 1 3)
do
  mkdir -p docker.local/miner$i/data/db/redis
  mkdir -p docker.local/miner$i/log
done

for i in $(seq 1 3)
do
  mkdir -p docker.local/sharder$i/data/blocks
  mkdir -p docker.local/sharder$i/data/cassandra
  mkdir -p docker.local/sharder$i/log
done
