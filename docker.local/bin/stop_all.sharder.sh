#!/bin/sh

for i in $(seq 1 3);
do
  SHARDER_ID=$i
  echo Stopping miner$SHARDER_ID ...
  SHARDER=$SHARDER_ID BLOCK_SIZE=${1:-5000} docker-compose -p miner$SHARDER_ID -f docker.local/build.miner/docker-compose.yml stop
done
