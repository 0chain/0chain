#!/bin/sh
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/ValidatorDockerfile . -t validator
docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/Dockerfile . -t blobber

for i in $(seq 1 6);
do
  BLOBBER=$i docker-compose -p blobber$i -f docker.local/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
