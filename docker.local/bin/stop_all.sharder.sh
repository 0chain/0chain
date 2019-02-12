#!/bin/sh

for i in $(seq 1 3);
do
  SHARDER_ID=$i
  echo Stopping sharder$SHARDER_ID ...
  SHARDER=$SHARDER_ID docker-compose -p sharder$SHARDER_ID -f docker.local/build.sharder/docker-compose.yml stop
done
