#!/bin/sh
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/build.unit_test/Dockerfile . -t unit_test

for i in $(seq 1 1);
do
  UNIT_TEST=$i docker-compose -p -f docker.local/build.unit_test/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
