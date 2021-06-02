#!/bin/bash

. ./paths.sh

cd $zChain_Root

for i in $(seq 1 2);
do
  MINER_ID=$i
  echo Stopping miner$MINER_ID ...
  MINER=$MINER_ID docker-compose -p miner$MINER_ID -f docker.local/build.miner/b0docker-compose.yml stop
done
