#!/bin/sh
docker-compose -p miner -f docker.aws/build.miner/docker-compose.yml build --force-rm
