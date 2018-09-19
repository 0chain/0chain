#!/bin/sh
echo Starting sharder...
docker-compose -p zchain -f docker.aws/build.sharder/docker-compose.yml up --detach

