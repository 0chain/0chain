#!/usr/bin/env bash
echo Starting miner...
docker-compose -p zchain -f /0chain/docker.aws/build.miner/docker-compose.yml up --detach

