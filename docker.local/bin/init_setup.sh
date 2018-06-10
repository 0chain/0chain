#!/bin/sh

for i in $(seq 1 3)
do
  mkdir -p docker.local/miner$i/data/db/redis
done

for i in $(seq 1 3)
do
  mkdir -p docker.local/sharder$i/data/blocks
done
