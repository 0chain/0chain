#!/bin/sh
docker-compose -p sharder -t zchain_sharder -f docker.aws/build.sharder/docker-compose.yml build --force-rm

