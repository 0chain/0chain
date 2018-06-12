#!/bin/sh
PWD=`pwd`
MINER_DIR=`basename $PWD`
MINER_ID=`echo $MINER_DIR | sed -e 's/.*\(.\)$/\1/'`

for i in $(seq 1 3);
do
  MINER_ID=$i
  echo Stopping miner$MINER_ID ...
  MINER=$MINER_ID BLOCK_SIZE=${1:-5000} docker-compose -p miner$MINER -f docker.local/build.miner/docker-compose.yml down
done
