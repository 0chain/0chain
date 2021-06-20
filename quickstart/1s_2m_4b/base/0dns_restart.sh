#!/bin/sh

. ./paths.sh

cd "$zDNS_Root" || exit
#------------------------------------------------------
cd ./docker.local/bin || exit

PWD=$(pwd)
echo Stopping 0dns ...
docker-compose -p 0dns -f ../docker-compose.yml stop

cd ../../

cd ./docker.local/bin || exit

PWD=$(pwd)
echo Starting 0dns ...
docker-compose -p 0dns -f ../docker-compose.yml up -d


# http://localhost:9091/
# http://198.18.0.98:9091/