#!/bin/sh
set -e

PWD=$(pwd)
SHARDER_DIR=$(basename "$PWD")
SHARDER_ID=$(echo "$SHARDER_DIR" | sed -e 's/.*\(.\)$/\1/')

SSD_PATH="${1:-..}"
HDD_PATH="${2:-..}"

echo Starting sharder"$SHARDER_ID" in daemon mode ...

SHARDER=$SHARDER_ID SSD_PATH=$PROJECT_ROOT_SSD HDD_PATH=$PROJECT_ROOT_HDD docker-compose -p sharder"$SHARDER_ID" -f ../build.sharder/p0docker-compose.yml up -d
