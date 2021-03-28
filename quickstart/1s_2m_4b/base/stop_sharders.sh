#!/bin/bash

. ./paths.sh

cd $zChain_Root

#----------------------------------------------

for i in $(seq 1 1);
do
  SHARDER_ID=$i
  echo Stopping sharder$SHARDER_ID ...
  SHARDER=$SHARDER_ID docker-compose -p sharder$SHARDER_ID -f docker.local/build.sharder/b0docker-compose.yml stop
done
