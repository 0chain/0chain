#!/bin/sh
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/build.miner/Dockerfile . -t miner

for i in $(seq 1 3);
do
  MINER=$i docker-compose -p miner$i -f docker.local/build.miner/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
