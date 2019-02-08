#!/bin/sh
docker build -f docker.local/build.sharder/Dockerfile . -t sharder

for i in $(seq 1 1);
do
  SHARDER=$i docker-compose -p sharder$i -f docker.local/build.sharder/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
