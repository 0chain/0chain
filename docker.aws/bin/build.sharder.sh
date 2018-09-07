#!/bin/sh
docker-compose -p zchain -t zchain_sharder -f docker.aws/build.sharder/docker-compose.yml build --force-rm

