#!/bin/bash

. ./paths.sh



#----------------------------------------------
# docker.local/bin/stop_all.sharder.sh


./stop_sharders.sh


#----------------------------------------------
cd $zChain_Root

sudo rm -rf docker.local/sharder*/log/*


#----------------------------------------------
# docker.local/bin/build.sharders.sh
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/build.sharder/Dockerfile . -t sharder

for i in $(seq 1 3);
do
  SHARDER=$i docker-compose -p sharder$i -f docker.local/build.sharder/b0docker-compose.yml build --force-rm
done

#docker.local/bin/sync_clock.sh
docker run --rm --privileged alpine hwclock -s

sleep 1

#----------------------------------------------


cd $zWorkflows_Base

./start_sharders.sh


