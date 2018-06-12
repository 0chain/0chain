#!/bin/sh
PWD=`pwd`
MINER_DIR=`basename $PWD`
MINER_ID=`echo $MINER_DIR | sed -e 's/.*\(.\)$/\1/'`


echo Starting miner$MINER_ID ...

MINER=$MINER_ID BLOCK_SIZE=${1:-5000} docker-compose -p miner$MINER_ID -f ../build.miner/docker-compose.yml up
