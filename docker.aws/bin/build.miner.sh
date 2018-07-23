#!/bin/sh
BLOCK_SIZE=5000 docker-compose -p miner -f docker.aws/build.miner/docker-compose.yml build --force-rm

