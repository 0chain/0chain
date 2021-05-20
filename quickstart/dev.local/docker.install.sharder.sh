#!/bin/bash
set -e

. ./env.sh



cd $zChain/docker.local/bin


for i in $(seq 1 1)
do
    echo ""
    echo "  - sharder$i"
    echo ""

    VOLUMES_CONFIG="$zCurrent/config/0chain" VOLUMES_DATA="$zCurrent/data" SHARDER=$i docker-compose -p sharder$i -f ../build.sharder/dev-docker-compose.yml up 
    #open http://127.0.0.1:717$i/_diagnostics

done








