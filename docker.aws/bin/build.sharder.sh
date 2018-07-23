#!/bin/sh
docker-compose -p sharder -f docker.aws/build.sharder/docker-compose.yml build --force-rm
