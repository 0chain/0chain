#!/bin/bash

. ./paths.sh

cd $zDNS_Root

#------------------------------------------------------

cd ./docker.local/bin

PWD=`pwd`
echo Stopping 0dns ...
docker-compose -p 0dns -f ../docker-compose.yml stop

cd ../../

sudo rm -rf ./docker.local/0dns/log/*
sudo rm -rf ./docker.local/0dns/mongodata/*


cd ./docker.local/bin

PWD=`pwd`
echo Starting 0dns ...
docker-compose -p 0dns -f ../docker-compose.yml up -d


# http://localhost:9091/
# http://198.18.0.98:9091/