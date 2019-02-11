#!/bin/sh
set -e

docker build -f docker.local/build.miner/Dockerfile . -t miner

for i in $(seq 1 3);
do
  MINER=$i BLOCK_SIZE=5000 docker-compose -p miner$i -f docker.local/build.miner/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
