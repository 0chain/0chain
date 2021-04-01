#!/bin/sh

for i in $(seq 1 4);
do
  MINER_ID=$i
  echo Stopping miner$MINER_ID ...
  MINER=$MINER_ID docker-compose -p miner$MINER_ID -f docker.local/build.miner/docker-compose.yml stop
done
