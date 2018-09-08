#!/usr/bin/env bash
echo Starting miner...
docker-compose -p zchain -u ubuntu -f docker.aws/build.miner/docker-compose.yml up --detach

