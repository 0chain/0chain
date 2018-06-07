#!/bin/sh

for i in $(seq 1 3);
do
  MINER=$i docker-compose -p miner$i -f docker.local/build.miner/docker-compose.yml build --force-rm
done
