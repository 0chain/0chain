#!/bin/bash

. ./paths.sh


./stop_miners.sh


#-------------------------------------------------------


sudo rm -rf docker.local/miner*/log/*

sleep 3

#-------------------------------------------------------

cd $zChain_Root


#./docker.local/bin/build.miners.sh

set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

DOCKERFILE="./docker.local/build.miner/Dockerfile"
sed 's,%COPY%,COPY --from=miner_build $APP_DIR,g' "$DOCKERFILE.template" > ./docker.local/build.miner/Dockerfile

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f ./docker.local/build.miner/Dockerfile . -t miner

for i in $(seq 1 3);
do
  MINER=$i docker-compose -p miner$i -f ./docker.local/build.miner/docker-compose.yml build --force-rm
done

#docker.local/bin/sync_clock.sh
docker run --rm --privileged alpine hwclock -s

#-------------------------------------------------------

sleep 3

cd $zWorkflows_Base

./start_miners.sh