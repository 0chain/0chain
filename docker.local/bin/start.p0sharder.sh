#!/bin/sh
set -e

PWD=$(pwd)
SHARDER_DIR=$(basename $PWD)
SHARDER_ID=$(echo "$SHARDER_DIR" | sed -e 's/.*\(.\)$/\1/')

echo Starting sharder"$SHARDER_ID" in daemon mode ...

SHARDER=$SHARDER_ID docker-compose -p sharder"$SHARDER_ID" -f ../build.sharder/p0docker-compose.yml up -d
