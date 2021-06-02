#!/bin/bash

. ./paths.sh

#-------------------------------------------------


./stop_blobbers.sh


#-------------------------------------------------

cd $zBlober_Root

sudo rm -rf ./docker.local/blobber*/log/*


#-------------------------------------------------
#./docker.local/bin/build.blobber.sh


set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/ValidatorDockerfile . -t validator
docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/Dockerfile . -t blobber

for i in $(seq 1 6);
do
  BLOBBER=$i docker-compose -p blobber$i -f docker.local/b0docker-compose.yml build --force-rm
done

#docker.local/bin/sync_clock.sh
docker run --rm --privileged alpine hwclock -s


#-------------------------------------------------

sleep 3

cd $zWorkflows_Base

./start_blobbers.sh
