#!/bin/sh

docker-compose -p zchain -f docker.local/docker-compose.yml build --force-rm
#docker-compose -p zchain -f docker.aws/build.miner/docker-compose.yml build --force-rm

