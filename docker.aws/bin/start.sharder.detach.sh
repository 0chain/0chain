#!/bin/sh
echo Starting sharder...
docker-compose -p zchain -f /0chain/docker.aws/build.sharder/docker-compose.yml up --detach

