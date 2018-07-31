#!/bin/sh

for i in $(seq 1 1);
do
  WALLET=$i docker-compose -p wallet$i -f docker.local/build.wallet/docker-compose.yml build --force-rm
done

