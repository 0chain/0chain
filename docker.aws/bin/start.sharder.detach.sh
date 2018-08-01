#!/bin/sh
echo Starting sharder...
docker-compose -p sharder -f docker.aws/build.sharder/docker-compose.yml up --detach

