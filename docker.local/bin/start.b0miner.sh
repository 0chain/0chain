#!/bin/sh
set -e

PWD=`pwd`
MINER_DIR=`basename $PWD`
MINER_ID=`echo $MINER_DIR | sed -e 's/.*\(.\)$/\1/'`

if [[ "$@" == *"--debug"* ]]
then
    echo Starting miner$MINER_ID in debug mode ...

    MINER=$MINER_ID docker-compose -p miner$MINER_ID -f ../build.miner/b0docker-compose-debug.yml up
else
    echo Starting miner$MINER_ID ...

    MINER=$MINER_ID docker-compose -p miner$MINER_ID -f ../build.miner/b0docker-compose.yml up
fi