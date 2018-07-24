#!/bin/sh
echo Starting sharder...$SHARDER
SHARDER=$SHARDER docker-compose -f docker.aws/build.sharder/docker-compose.yml up

