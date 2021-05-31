#!/bin/bash

# !!! start.b0sharder.sh - For now just a single sharder.

. ./paths.sh

cd $zChain_Root
#----------------------------------------------

cd ./docker.local/sharder1

PWD=`pwd`
SHARDER_DIR=`basename $PWD`
SHARDER_ID=`echo $SHARDER_DIR | sed -e 's/.*\(.\)$/\1/'`

echo Starting sharder$SHARDER_ID ...

SHARDER=$SHARDER_ID docker-compose -p sharder$SHARDER_ID -f ../build.sharder/b0docker-compose.yml up &