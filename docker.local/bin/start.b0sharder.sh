#!/bin/bash
set -e

PWD=`pwd`
SHARDER_DIR=`basename $PWD`
SHARDER_ID=`echo $SHARDER_DIR | sed -e 's/.*\(.\)$/\1/'`

if [[ "$@" == *"--debug"* ]]
then
    echo Starting sharder$SHARDER_ID in debug mode ...

    SHARDER=$SHARDER_ID docker-compose -p sharder$SHARDER_ID -f ../build.sharder/b0docker-compose-debug.yml up
else
    echo Starting sharder$SHARDER_ID ...

    SHARDER=$SHARDER_ID docker-compose -p sharder$SHARDER_ID -f ../build.sharder/b0docker-compose.yml up
fi
