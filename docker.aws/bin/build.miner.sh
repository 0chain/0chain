#!/bin/sh
docker-compose -p miner -t zchain_miner -f docker.aws/build.miner/docker-compose.yml build --force-rm
