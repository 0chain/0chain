#!/bin/sh
PWD=$(pwd)
NODE_DIR=$(basename "$PWD")
NODE_ID=$(echo "$NODE_DIR" | sed -e 's/.*\(.\)$/\1/')

SERVICE=$1; shift
CMD=$1; shift

SHARD_NO="${NODE_DIR//[^0-9]/}"

echo "$NODE_DIR: running $SERVICE $CMD $*"

SHARDER=$SHARD_NO docker-compose -p "$NODE_DIR" -f ../build.sharder/docker-compose.yml exec $SERVICE $CMD $*
