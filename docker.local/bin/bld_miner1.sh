#!/bin/sh

  MINER=1 BLOCK_SIZE=5000 docker-compose -p miner1 -f docker.local/build.miner/docker-compose.yml build --force-rm

docker.local/bin/sync_clock.sh
