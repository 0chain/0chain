#!/bin/bash
set -e


. ./env.sh

cd $zChain

echo ">"
echo ">"
echo "> setup testnet0 in docker's network"
docker network ls | grep testnet0 || docker network create --driver=bridge --subnet=198.18.0.0/15 --gateway=198.18.0.255 testnet0

echo ">"
echo ">"
echo "> build images: zchain_build_base & zchain_run_base"
./docker.local/bin/build.base.sh 

echo ">"
echo ">"
echo "> build image: miner"
./docker.local/bin/build.miners.sh

echo ">"
echo ">"
echo "> build image: sharder"
docker build --build-arg GIT_COMMIT="dev" -f docker.local/build.sharder/Dockerfile . -t sharder

echo ">"
echo ">"
echo "> build image: 0dns"
cd $zDNS
VOLUMES_CONFIG="$zCurrent/config/0dns" VOLUMES_LOG="$zCurrent/data/0dns/log" VOLUMES_MONGO_DATA="$zCurrent/data/0dns/mongodata"  docker-compose -p 0dns -f docker.local/docker-compose.yml up -d

echo ">"
echo ">"
echo "> build images: blobber & validator"
cd $zBlobber
#./docker.local/bin/build.blobber.sh 