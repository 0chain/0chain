#!/bin/sh
PWD=$(pwd)
SHARDER_DIR=$(basename "$PWD")
SHARDER_ID=$(echo "$SHARDER_DIR" | sed -e 's/.*\(.\)$/\1/')

echo Stopping sharder"$SHARDER_ID" ...

SHARDER=$SHARDER_ID docker-compose -p sharder"$SHARDER_ID" -f ../build.sharder/b0docker-compose.yml stop
