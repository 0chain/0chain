#!/bin/sh
BLOCK_SIZE=${1:-5000} docker-compose -f docker.aws/build.sharder/docker-compose.yml stop
