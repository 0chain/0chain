#!/bin/sh

for i in $(seq 1 3);
do
  MINER_ID=$i
  echo Stopping miner$MINER_ID ...
  MINER=$MINER_ID BLOCK_SIZE=${1:-5000} docker-compose -p miner$MINER_ID -f docker.local/build.miner/docker-compose.yml stop
done
