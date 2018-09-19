#!/usr/bin/env bash
echo Starting miner...
docker-compose -p zchain -f docker.aws/build.miner/docker-compose.yml up

