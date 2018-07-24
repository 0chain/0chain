#!/usr/bin/env bash
echo Starting miner...$MINER
docker-compose -p miner -f docker.aws/build.miner/docker-compose.yml up

