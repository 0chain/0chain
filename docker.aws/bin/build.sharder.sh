#!/bin/sh
docker-compose -p zchain -f docker.aws/build.sharder/docker-compose.yml build --force-rm

